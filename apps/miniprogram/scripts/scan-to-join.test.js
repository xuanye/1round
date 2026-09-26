const assert = require("assert");

const { loadSource } = require('./page-test-utils');

let page;
let scanResult;
let navigatedTo;
let toast;
const wx = {
  scanCode({ success }) { success(scanResult); },
  navigateTo({ url }) { navigatedTo = url; },
  showToast({ title }) { toast = title; },
};
page = loadSource('pages/home/index.ts', { wx }).page;

for (const [label, result] of [
  ["unescaped scene", { path: "pages/game-join/index?scene=code=ABC123" }],
  ["encoded scene", { path: "pages/game-join/index?scene=code%3DABC123" }],
  ["direct invite link", { result: "pages/game-join/index?inviteCode=ABC123" }],
]) {
  scanResult = result;
  navigatedTo = undefined;
  toast = undefined;
  page.scanToJoin();
  assert.strictEqual(navigatedTo, "/pages/game-join/index?inviteCode=ABC123", label);
  assert.strictEqual(toast, undefined, label);
}
