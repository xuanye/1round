const assert = require("assert");
const fs = require("fs");
const path = require("path");
const vm = require("vm");
const ts = require("typescript");

const projectRoot = path.resolve(__dirname, "..");
const source = fs.readFileSync(path.join(projectRoot, "src/pages/home/index.ts"), "utf8");
const compiled = ts.transpileModule(source, {
  compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2019 },
}).outputText;

let page;
let scanResult;
let navigatedTo;
let toast;
const wx = {
  scanCode({ success }) { success(scanResult); },
  navigateTo({ url }) { navigatedTo = url; },
  showToast({ title }) { toast = title; },
};
vm.runInNewContext(compiled, {
  Page(value) { page = value; },
  wx,
  exports: {},
  require() { return {}; },
});

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
