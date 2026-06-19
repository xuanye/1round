const fs = require("fs");
const path = require("path");
const { spawn, spawnSync } = require("child_process");

const projectRoot = path.resolve(__dirname, "..");
const srcRoot = path.join(projectRoot, "src");
const distRoot = path.join(projectRoot, "dist");
const tscBin = require.resolve("typescript/bin/tsc");
const watchMode = process.argv.includes("--watch");

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
  process.exit(0);
}

const tsc = startTypeScriptWatch();

fs.watch(srcRoot, { recursive: true }, (_eventType, filename) => {
  syncChangedAsset(filename);
});

tsc.on("exit", (code) => {
  process.exit(code || 0);
});
