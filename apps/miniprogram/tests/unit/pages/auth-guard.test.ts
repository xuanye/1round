export {};

const fs = require('node:fs');
const path = require('node:path');

const srcRoot = path.join(__dirname, '../../../src');

const protectedPages = [
  'pages/home/index.ts',
  'pages/game-create/index.ts',
  'pages/game-join/index.ts',
  'pages/player-manage/index.ts',
  'pages/score-input/index.ts',
  'pages/ranking/index.ts',
  'pages/history/index.ts',
  'pages/mine/index.ts',
];

describe('auth guard', () => {
  it('requires login on every protected page', () => {
    for (const page of protectedPages) {
      const filePath = path.join(srcRoot, page);
      const source = fs.readFileSync(filePath, 'utf8');
      expect(source).toMatch(/requireLogin/);
      expect(source).toMatch(/from ['"]\.\.\/\.\.\/services\/auth\.service['"]/);
    }
  });

  it('checks shareToken before requireLogin in game-detail', () => {
    const gameDetail = fs.readFileSync(path.join(srcRoot, 'pages/game-detail/page.ts'), 'utf8');
    expect(gameDetail).toMatch(/requireLogin/);
    expect(gameDetail).toMatch(/this\.data\.shareToken/);
    expect(gameDetail.indexOf('this.data.shareToken')).toBeLessThan(
      gameDetail.lastIndexOf('requireLogin'),
    );
    expect(gameDetail).toMatch(/getPublicSettlement/);
  });
});
