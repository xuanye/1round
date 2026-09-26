// QR renderer smoke test: builds, then exercises the compiled dist module
// against a stubbed canvas context. Ported 1:1 from scripts/qrcode.test.js.
export {};

const { spawnSync } = require('node:child_process');
const path = require('node:path');

const projectRoot = path.resolve(__dirname, '../..');

jest.setTimeout(120000);

function runBuild() {
  const result = spawnSync(process.execPath, ['scripts/build.js'], {
    cwd: projectRoot,
    encoding: 'utf8',
  });
  expect(result.status).toBe(0);
}

describe('qrcode smoke', () => {
  it('draws visible modules through the compiled build', () => {
    runBuild();

    const fillRects: Array<{ x: number; y: number; width: number; height: number }> = [];
    (globalThis as Record<string, unknown>).wx = {
      createCanvasContext() {
        return {
          setFillStyle() {},
          fillRect(x: number, y: number, width: number, height: number) {
            fillRects.push({ x, y, width, height });
          },
          draw() {},
        };
      },
    };

    const { drawQRCode } = require(path.join(projectRoot, 'dist', 'utils', 'qrcode.js'));
    expect(() => drawQRCode('inviteQR', 'https://oneround.app/join?code=ABC123', 180, {})).not.toThrow();
    expect(fillRects.length).toBeGreaterThan(20);
  });
});
