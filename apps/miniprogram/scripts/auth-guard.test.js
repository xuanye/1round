const assert = require("assert");
const fs = require("fs");
const path = require("path");

const projectRoot = path.resolve(__dirname, "..");
const srcRoot = path.join(projectRoot, "src");

const protectedPages = [
  "pages/home/index.ts",
  "pages/game-create/index.ts",
  "pages/game-join/index.ts",
  "pages/player-manage/index.ts",
  "pages/score-input/index.ts",
  "pages/ranking/index.ts",
  "pages/history/index.ts",
];

for (const page of protectedPages) {
  const filePath = path.join(srcRoot, page);
  const source = fs.readFileSync(filePath, "utf8");
  assert.match(source, /requireLogin/, `${page} must call requireLogin()`);
  assert.match(
    source,
    /from ['"]\.\.\/\.\.\/services\/auth\.service['"]/,
    `${page} must import requireLogin from auth.service`,
  );
}

const gameDetail = fs.readFileSync(path.join(srcRoot, "pages/game-detail/index.ts"), "utf8");
assert.match(gameDetail, /requireLogin/, "game-detail protected mode must call requireLogin()");
assert.match(gameDetail, /this\.data\.shareToken/, "game-detail must preserve shareToken public mode");
assert(
  gameDetail.indexOf("this.data.shareToken") < gameDetail.lastIndexOf("requireLogin"),
  "game-detail must check shareToken before requireLogin()",
);
assert.match(gameDetail, /getPublicSettlement/, "game-detail must keep public settlement loading");
