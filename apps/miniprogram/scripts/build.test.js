const assert = require("assert");
const fs = require("fs");
const path = require("path");
const { spawn } = require("child_process");

const projectRoot = path.resolve(__dirname, "..");
const envPath = path.join(projectRoot, ".env");
const localEnvPath = path.join(projectRoot, ".env.local");
const configOutputPath = path.join(projectRoot, "dist", "utils", "config.js");
const appJsPath = path.join(projectRoot, "dist", "app.js");

function readIfExists(filePath) {
  return fs.existsSync(filePath) ? fs.readFileSync(filePath, "utf8") : null;
}

function restore(filePath, content) {
  if (content === null) {
    fs.rmSync(filePath, { force: true });
    return;
  }
  fs.writeFileSync(filePath, content);
}

// Runs a full build while polling dist/app.js: the entry JS must never
// disappear, otherwise WeChat DevTools loads a JS-less dist and reports
// a load failure (removeDist window).
function runBuildWatchingEntry() {
  return new Promise((resolve, reject) => {
    fs.mkdirSync(path.dirname(appJsPath), { recursive: true });
    fs.writeFileSync(appJsPath, "/* sentinel */");

    const child = spawn(process.execPath, ["scripts/build.js"], {
      cwd: projectRoot,
      encoding: "utf8",
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

const originalEnv = readIfExists(envPath);
const originalLocalEnv = readIfExists(localEnvPath);

(async () => {
  try {
    fs.writeFileSync(envPath, "ONEROUND_API_BASE_URL=https://1round.xuanye.wang\n");
    fs.writeFileSync(localEnvPath, "ONEROUND_API_BASE_URL=http://127.0.0.1:19090\n");

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
    restore(envPath, originalEnv);
    restore(localEnvPath, originalLocalEnv);
  }
})().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
