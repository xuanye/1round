export {};

const fs = require('node:fs');
const path = require('node:path');
const simulate = require('miniprogram-simulate');

// Render the actual shared template: permissions must affect visible controls.
const componentId = simulate.load({
  template: fs.readFileSync(path.resolve(__dirname, '../../src/pages/game-detail/content.wxml'), 'utf8'),
  data: {
    isHome: true, isPublicShare: false, game: { name: '测试牌局', status: 'active', isCreator: true },
    icons: {}, participants: [{ id: 'p1', name: 'Alice', isMe: true, isCreator: true, score: '0' }],
    transfers: [], visibleTransfers: [], canExit: true,
  },
  methods: { none() {}, renameSelf() {}, openRanking() {}, exitGame() {}, finish() {} },
});

function renderGame() {
  const component = simulate.render(componentId);
  component.attach(document.createElement('parent-wrapper'));
  return component;
}

describe('active game controls', () => {
  it('renders creator exit, role, rename hint, and separate ranking entry', () => {
    const component = renderGame();
    const actions = component.querySelectorAll('.quiet-action');
    expect(actions.map((item: any) => item.dom.textContent)).toEqual(['结束本局', '退出牌局']);
    expect(actions[1].toJSON().attrs).toContainEqual({ name: 'disabled', value: false });
    expect(component.querySelector('.score-player-self').dom.textContent).toContain('创建者');
    expect(component.querySelector('.score-rename-hint').dom.textContent).toBe('点击修改昵称');
    expect(component.querySelector('.section-link').dom.textContent).toContain('本局排名');
  });

  it('keeps nonzero exit disabled and renders an explanation for all roles', () => {
    const component = renderGame();
    for (const isCreator of [true, false]) {
      component.setData({ canExit: false, game: { name: '测试牌局', status: 'active', isCreator } });
      const actions = component.querySelectorAll('.quiet-action');
      expect(actions[1].dom.textContent).toBe('退出牌局');
      expect(actions[1].toJSON().attrs).toContainEqual({ name: 'disabled', value: true });
      expect(component.querySelectorAll('.scoring-hint').some((item: any) =>
        item.dom.textContent.includes('当前分值不为 0'))).toBe(true);
    }
  });
});
