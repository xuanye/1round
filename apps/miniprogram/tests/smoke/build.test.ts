// Build smoke test: runs the real build twice (sentinel-watch + clean) and
// verifies the dist entry never disappears and the API base URL override is
// baked in. Ported 1:1 from scripts/build.test.js.
export {};

const fs = require('node:fs');
const path = require('node:path');
const { spawn, spawnSync } = require('node:child_process');

const projectRoot = path.resolve(__dirname, '../..');
const configOutputPath = path.join(projectRoot, 'dist', 'utils', 'config.js');
const appJsPath = path.join(projectRoot, 'dist', 'app.js');

jest.setTimeout(120000);

// Runs a full build while polling dist/app.js: the entry JS must never
// disappear, otherwise WeChat DevTools loads a JS-less dist and reports
// a load failure (removeDist window). The API base URL override is passed
// via process.env instead of .env files so tests never touch local config.
function runBuildWatchingEntry() {
  return new Promise<{ code: number | null; stdout: string; stderr: string; polls: number; missing: number }>(
    (resolve, reject) => {
      fs.mkdirSync(path.dirname(appJsPath), { recursive: true });
      fs.writeFileSync(appJsPath, '/* sentinel */');

      const child = spawn(process.execPath, ['scripts/build.js'], {
        cwd: projectRoot,
        encoding: 'utf8',
        env: { ...process.env, ONEROUND_API_BASE_URL: 'http://127.0.0.1:19090' },
      });

      let stdout = '';
      let stderr = '';
      let polls = 0;
      let missing = 0;
      const timer = setInterval(() => {
        polls += 1;
        if (!fs.existsSync(appJsPath)) missing += 1;
      }, 10);

      child.stdout.on('data', (chunk: string) => { stdout += chunk; });
      child.stderr.on('data', (chunk: string) => { stderr += chunk; });
      child.on('error', (error: Error) => {
        clearInterval(timer);
        reject(error);
      });
      child.on('exit', (code: number | null) => {
        clearInterval(timer);
        resolve({ code, stdout, stderr, polls, missing });
      });
    },
  );
}

// Rebuilds without any override so a finished test run leaves dist
// pointing at the production API, never at a local debug address.
function runCleanBuild() {
  const result = spawnSync(process.execPath, ['scripts/build.js'], {
    cwd: projectRoot,
    encoding: 'utf8',
    env: (() => {
      const env: Record<string, string | undefined> = { ...process.env };
      delete env.ONEROUND_API_BASE_URL;
      return env;
    })(),
  });
  if (result.status !== 0) {
    throw new Error(`clean rebuild failed\nstdout:\n${result.stdout}\nstderr:\n${result.stderr}`);
  }
}

describe('build smoke', () => {
  it('keeps dist/app.js present throughout the build and applies the API override', async () => {
    try {
      const { code, stdout, stderr, polls, missing } = await runBuildWatchingEntry();
      if (code !== 0) {
        throw new Error(`build failed\nstdout:\n${stdout}\nstderr:\n${stderr}`);
      }
      expect(polls).toBeGreaterThan(0);
      expect(missing).toBe(0);

      const output = fs.readFileSync(configOutputPath, 'utf8');
      expect(output).toMatch(/http:\/\/127\.0\.0\.1:19090/);
      expect(output).not.toMatch(/__ONEROUND_API_BASE_URL__/);

      if (require.cache) delete require.cache[require.resolve(configOutputPath)];
      const { getBaseUrl } = require(configOutputPath);
      expect(getBaseUrl()).toBe('http://127.0.0.1:19090');
    } finally {
      runCleanBuild();
    }
  });
});
