const assert = require("assert");
const { spawnSync } = require("child_process");
const path = require("path");

const projectRoot = path.resolve(__dirname, "..");

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

runBuild();

const fillRects = [];
global.wx = {
  createCanvasContext() {
    return {
      setFillStyle() {},
      fillRect(x, y, width, height) {
        fillRects.push({ x, y, width, height });
      },
      draw() {},
    };
  },
};

const { drawQRCode } = require(path.join(projectRoot, "dist", "utils", "qrcode.js"));

assert.doesNotThrow(() => {
  drawQRCode("inviteQR", "https://oneround.app/join?code=ABC123", 180, {});
});

assert.ok(fillRects.length > 20, "expected QR renderer to draw visible modules");
