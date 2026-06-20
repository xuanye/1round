const assert = require("assert");
const fs = require("fs");
const path = require("path");
const { spawnSync } = require("child_process");

const projectRoot = path.resolve(__dirname, "..");
const envPath = path.join(projectRoot, ".env");
const localEnvPath = path.join(projectRoot, ".env.local");
const configOutputPath = path.join(projectRoot, "dist", "utils", "config.js");

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

function runBuild() {
  const result = spawnSync(process.execPath, ["scripts/build.js"], {
    cwd: projectRoot,
    encoding: "utf8",
  });

  assert.strictEqual(
    result.status,
    0,
    `build failed\nstdout:\n${result.stdout}\nstderr:\n${result.stderr}`,
  );
}

const originalEnv = readIfExists(envPath);
const originalLocalEnv = readIfExists(localEnvPath);

try {
  fs.writeFileSync(envPath, "ONEROUND_API_BASE_URL=https://1round.xuanye.wang\n");
  fs.writeFileSync(localEnvPath, "ONEROUND_API_BASE_URL=http://127.0.0.1:19090\n");

  runBuild();

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
