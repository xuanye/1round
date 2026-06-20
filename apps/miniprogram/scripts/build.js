const fs = require("fs");
const path = require("path");
const { spawn, spawnSync } = require("child_process");

const projectRoot = path.resolve(__dirname, "..");
const srcRoot = path.join(projectRoot, "src");
const distRoot = path.join(projectRoot, "dist");
const tscBin = require.resolve("typescript/bin/tsc");
const watchMode = process.argv.includes("--watch");
const apiBaseUrlToken = "__ONEROUND_API_BASE_URL__";
const defaultApiBaseUrl = "https://1round.xuanye.wang";

function parseEnvFile(filePath) {
  if (!fs.existsSync(filePath)) {
    return {};
  }

  return fs
    .readFileSync(filePath, "utf8")
    .split(/\r?\n/)
    .reduce((values, line) => {
      const trimmed = line.trim();
      if (!trimmed || trimmed.startsWith("#")) {
        return values;
      }

      const separatorIndex = trimmed.indexOf("=");
      if (separatorIndex < 0) {
        return values;
      }

      const key = trimmed.slice(0, separatorIndex).trim();
      let value = trimmed.slice(separatorIndex + 1).trim();
      if (
        (value.startsWith('"') && value.endsWith('"')) ||
        (value.startsWith("'") && value.endsWith("'"))
      ) {
        value = value.slice(1, -1);
      }

      values[key] = value;
      return values;
    }, {});
}

function loadBuildEnv() {
  return {
    ...parseEnvFile(path.join(projectRoot, ".env")),
    ...parseEnvFile(path.join(projectRoot, ".env.local")),
    ...process.env,
  };
}

function apiBaseUrl() {
  return loadBuildEnv().ONEROUND_API_BASE_URL || defaultApiBaseUrl;
}

function isTypeScriptFile(filePath) {
  return filePath.endsWith(".ts") || filePath.endsWith(".tsx");
}

function removeDist() {
  fs.rmSync(distRoot, { recursive: true, force: true });
  fs.mkdirSync(distRoot, { recursive: true });
}

function copyStaticAssets(sourceDir = srcRoot) {
  for (const entry of fs.readdirSync(sourceDir, { withFileTypes: true })) {
    const sourcePath = path.join(sourceDir, entry.name);
    const relativePath = path.relative(srcRoot, sourcePath);
    const targetPath = path.join(distRoot, relativePath);

    if (entry.isDirectory()) {
      fs.mkdirSync(targetPath, { recursive: true });
      copyStaticAssets(sourcePath);
      continue;
    }

    if (entry.isFile() && !isTypeScriptFile(sourcePath)) {
      fs.mkdirSync(path.dirname(targetPath), { recursive: true });
      fs.copyFileSync(sourcePath, targetPath);
    }
  }
}

function runTypeScriptBuild() {
  const result = spawnSync(process.execPath, [tscBin, "-p", "tsconfig.json"], {
    cwd: projectRoot,
    stdio: "inherit",
  });

  if (result.status !== 0) {
    process.exit(result.status || 1);
  }
}

function injectRuntimeConfig(sourceDir = distRoot) {
  if (!fs.existsSync(sourceDir)) {
    return;
  }

  const replacement = apiBaseUrl();
  for (const entry of fs.readdirSync(sourceDir, { withFileTypes: true })) {
    const targetPath = path.join(sourceDir, entry.name);
    if (entry.isDirectory()) {
      injectRuntimeConfig(targetPath);
      continue;
    }

    if (
      !entry.isFile() ||
      (!targetPath.endsWith(".js") && !targetPath.endsWith(".js.map"))
    ) {
      continue;
    }

    const current = fs.readFileSync(targetPath, "utf8");
    const next = current.replaceAll(apiBaseUrlToken, replacement);
    if (next !== current) {
      fs.writeFileSync(targetPath, next);
    }
  }
}

function startTypeScriptWatch() {
  return spawn(
    process.execPath,
    [tscBin, "-p", "tsconfig.json", "--watch", "--preserveWatchOutput"],
    {
      cwd: projectRoot,
      stdio: "inherit",
    },
  );
}

function syncChangedAsset(filename) {
  if (!filename || isTypeScriptFile(filename)) {
    return;
  }

  const sourcePath = path.join(srcRoot, filename);
  const targetPath = path.join(distRoot, filename);

  if (!fs.existsSync(sourcePath)) {
    fs.rmSync(targetPath, { recursive: true, force: true });
    return;
  }

  const stat = fs.statSync(sourcePath);
  if (stat.isDirectory()) {
    copyStaticAssets(sourcePath);
    return;
  }

  fs.mkdirSync(path.dirname(targetPath), { recursive: true });
  fs.copyFileSync(sourcePath, targetPath);
}

removeDist();
copyStaticAssets();

if (!watchMode) {
  runTypeScriptBuild();
  injectRuntimeConfig();
  process.exit(0);
}

const tsc = startTypeScriptWatch();

fs.watch(srcRoot, { recursive: true }, (_eventType, filename) => {
  syncChangedAsset(filename);
});

fs.watch(distRoot, { recursive: true }, (_eventType, filename) => {
  if (filename && (filename.endsWith(".js") || filename.endsWith(".js.map"))) {
    injectRuntimeConfig(path.dirname(path.join(distRoot, filename)));
  }
});

tsc.on("exit", (code) => {
  process.exit(code || 0);
});
