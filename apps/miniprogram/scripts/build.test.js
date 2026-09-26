const assert = require("assert");
const fs = require("fs");
const path = require("path");
const { spawn } = require("child_process");

const projectRoot = path.resolve(__dirname, "..");
const configOutputPath = path.join(projectRoot, "dist", "utils", "config.js");
const appJsPath = path.join(projectRoot, "dist", "app.js");

// Runs a full build while polling dist/app.js: the entry JS must never
// disappear, otherwise WeChat DevTools loads a JS-less dist and reports
// a load failure (removeDist window). The API base URL override is passed
// via process.env instead of .env files so tests never touch local config.
function runBuildWatchingEntry() {
  return new Promise((resolve, reject) => {
    fs.mkdirSync(path.dirname(appJsPath), { recursive: true });
    fs.writeFileSync(appJsPath, "/* sentinel */");

    const child = spawn(process.execPath, ["scripts/build.js"], {
      cwd: projectRoot,
      encoding: "utf8",
      env: { ...process.env, ONEROUND_API_BASE_URL: "http://127.0.0.1:19090" },
    });

    let stdout = "";
    let stderr = "";
    let polls = 0;
    let missing = 0;
    const timer = setInterval(() => {
      polls += 1;
      if (!fs.existsSync(appJsPath)) missing += 1;
    }, 10);

    child.stdout.on("data", (chunk) => { stdout += chunk; });
    child.stderr.on("data", (chunk) => { stderr += chunk; });
    child.on("error", (error) => {
      clearInterval(timer);
      reject(error);
    });
    child.on("exit", (code) => {
      clearInterval(timer);
      resolve({ code, stdout, stderr, polls, missing });
    });
  });
}

// Rebuilds without any override so a finished test run leaves dist
// pointing at the production API, never at a local debug address.
function runCleanBuild() {
  const result = require("child_process").spawnSync(process.execPath, ["scripts/build.js"], {
    cwd: projectRoot,
    encoding: "utf8",
    env: (() => {
      const env = { ...process.env };
      delete env.ONEROUND_API_BASE_URL;
      return env;
    })(),
  });
  assert.strictEqual(
    result.status,
    0,
    `clean rebuild failed\nstdout:\n${result.stdout}\nstderr:\n${result.stderr}`,
  );
}

(async () => {
  try {
    const { code, stdout, stderr, polls, missing } = await runBuildWatchingEntry();
    assert.strictEqual(
      code,
      0,
      `build failed\nstdout:\n${stdout}\nstderr:\n${stderr}`,
    );
    assert.ok(polls > 0, "build finished too fast to poll dist/app.js");
    assert.strictEqual(
      missing,
      0,
      `dist/app.js disappeared in ${missing}/${polls} polls during build (load window)`,
    );

    const output = fs.readFileSync(configOutputPath, "utf8");
    assert.match(output, /http:\/\/127\.0\.0\.1:19090/);
    assert.doesNotMatch(output, /__ONEROUND_API_BASE_URL__/);

    delete require.cache[require.resolve(configOutputPath)];
    const { getBaseUrl } = require(configOutputPath);
    assert.strictEqual(getBaseUrl(), "http://127.0.0.1:19090");
  } finally {
    runCleanBuild();
  }
})().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
