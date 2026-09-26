const assert = require('assert');
const fs = require('fs');
const path = require('path');
const { loadSource, instance } = require('./page-test-utils');

const user = { id: 'user-1', displayName: 'Alice' };
function harness() {
  const state = {
    current: { id: 'game-1', inviteCode: 'ABC123' },
    score: 0, summaryCalls: 0, historyCalls: 0, logins: 0, sockets: [], routes: [],
  };
  class Realtime {
    constructor() { state.sockets.push(this); }
    connect(id) { this.id = id; }
    onEvent(handler) { this.handler = handler; }
    disconnect() { this.disconnected = true; }
  }
  const wx = {
    showLoading() {}, hideLoading() {}, showToast() {}, stopPullDownRefresh() {},
    switchTab({ url }) { state.routes.push(['tab', url]); },
    navigateTo({ url }) { state.routes.push(['page', url]); },
    showModal(options) { state.modal = options; },
  };
  const globals = {
    wx,
    require(name) {
      if (name.includes('auth.service')) return { requireLogin: async () => { state.logins++; return user; } };
      if (name.includes('storage')) return { getUser: () => user, saveRecentSession() {} };
      if (name.includes('format')) return { formatScore: n => n > 0 ? `+${n}` : String(n), formatFriendlyTime: () => '昨天' };
      if (name.includes('realtime.service')) return { RealtimeService: Realtime };
      if (name.includes('game.service')) return {
        getCurrentGame: async () => {
          if (state.failCurrent) throw new Error('离线');
          return state.current;
        },
        getSummary: async () => {
          state.summaryCalls++;
          return {
            name: 'Test game', status: 'active', ownerUserId: user.id, inviteCode: 'ABC123',
            updatedAt: '2026-09-26T00:00:00Z',
            players: [
              { id: 'p1', userId: user.id, displayName: 'Alice', totalScore: state.score },
              { id: 'p2', userId: 'user-2', displayName: 'Bob', totalScore: -state.score },
            ],
            roundStatus: null,
          };
        },
        getScoreTransfers: async () => [],
        getHistory: async () => { state.historyCalls++; return { items: [] }; },
        leaveGame: async () => { state.current = null; },
      };
      return {};
    },
  };
  return { state, globals, home: instance(loadSource('pages/home/index.ts', globals).page) };
}

async function testCustomTabSelection() {
  const { home, globals, state } = harness();
  const selected = [];
  home.getTabBar = () => ({ setData: data => selected.push(data.selected) });
  await home.onShow();
  assert.equal(selected.pop(), 0, 'home restores its selected tab when shown');

  const mine = instance(loadSource('pages/mine/index.ts', globals).page);
  mine.getTabBar = () => ({ setData: data => selected.push(data.selected) });
  await mine.onShow();
  assert.equal(selected.pop(), 2, 'programmatic navigation restores the mine selection');

  let component;
  loadSource('custom-tab-bar/index.ts', Object.assign({}, globals, {
    Component(value) { component = value; },
  }));
  const tabBar = { data: component.data };
  const select = index => component.methods.selectTab.call(tabBar, { currentTarget: { dataset: { index } } });
  select(0);
  select(99);
  assert.equal(state.routes.length, 0, 'current or invalid tabs do not navigate');
  select(1);
  assert.deepEqual(state.routes.pop(), ['tab', '/pages/ranking/index']);
  select(2);
  assert.deepEqual(state.routes.pop(), ['tab', '/pages/mine/index']);
}

async function testActiveHome() {
  const { home, state } = harness();
  await home.onShow();
  assert.equal(home.data.homeState, 'ready');
  assert.equal(home.data.id, 'game-1');
  assert.equal(home.data.participants.length, 2);
  assert.equal(state.historyCalls, 0, 'active home does not load historical cards');
  home.inputScore();
  assert.deepEqual(state.routes.pop(), ['page', '/pages/score-input/index?id=game-1']);
  state.score = 20;
  await state.sockets[0].handler();
  // The event handler schedules an HTTP refresh; allow its awaited operations to finish.
  await new Promise(resolve => setImmediate(resolve));
  assert.equal(home.data.participants[0].score, '+20');
  home.onHide();
  assert.equal(state.sockets[0].disconnected, true);
  await home.onShow();
  assert.equal(state.sockets.length, 2, 'returning to the tab reconnects with fresh summary');
  home.exitGame();
  assert.equal(state.modal, undefined, 'nonzero participant cannot exit');
  state.score = 0;
  await home.refreshHome();
  home.exitGame();
  await state.modal.success({ confirm: true });
  assert.equal(home.data.id, '');
  assert.equal(home.data.homeState, 'ready');
  assert.equal(state.historyCalls, 1);
}

async function testEmptyAndOffline() {
  const { home, state } = harness();
  state.current = null;
  await home.onShow();
  assert.equal(home.data.id, '');
  assert.equal(state.summaryCalls, 0);
  home.createGame();
  assert.deepEqual(state.routes.pop(), ['page', '/pages/game-create/index']);
  state.failCurrent = true;
  await home.onShow();
  assert.equal(home.data.homeState, 'error');
  assert.equal(home.data.homeError, '离线');
  state.failCurrent = false;
  await home.refreshHome();
  assert.equal(home.data.homeState, 'ready');
}

async function testDetailRoutesAndPublicAccess() {
  const { globals, state } = harness();
  const page = instance(loadSource('pages/game-detail/index.ts', globals).page);
  await page.onLoad({ id: 'game-1' });
  await page.onShow();
  assert.deepEqual(state.routes.pop(), ['tab', '/pages/home/index']);
  const publicPage = instance(loadSource('pages/game-detail/index.ts', globals).page);
  let publicLoads = 0;
  publicPage.loadPublicSettlement = async () => { publicLoads++; };
  await publicPage.onLoad({ shareToken: 'public-token' });
  const loginsBefore = state.logins;
  await publicPage.onShow();
  assert.equal(publicLoads, 1);
  assert.equal(state.logins, loginsBefore, 'public settlement must not trigger login');
}

async function testCreateAndJoinReturnToTab() {
  for (const file of ['game-create', 'game-join']) {
    let destination;
    const globals = {
      wx: {
        showLoading() {}, hideLoading() {}, showToast() {},
        switchTab({ url }) { destination = url; },
      },
      require(name) {
        if (name.includes('auth.service')) return { requireLogin: async () => user };
        if (name.includes('game.service')) return {
          createGame: async () => ({ id: 'new-game', inviteCode: 'ABC123' }),
          joinGame: async () => ({ gameSessionId: 'new-game' }),
        };
        return { saveRecentSession() {} };
      },
    };
    const page = instance(loadSource(`pages/${file}/index.ts`, globals).page);
    page.data.displayName = 'Alice';
    await page.submit();
    assert.equal(destination, '/pages/home/index', `${file} must return to the game tab`);
  }
}

(async () => {
  for (const [device, expected] of [
    [{ platform: 'ios', system: 'iOS 18' }, 'font-ios'],
    [{ platform: 'android', system: 'Android 15', brand: 'HUAWEI' }, 'font-android'],
    [{ platform: 'ohos', system: 'HarmonyOS 5' }, 'font-harmony'],
    [{ platform: 'android', system: 'HarmonyOS 4' }, 'font-harmony'],
    [{ platform: 'devtools', system: 'Unknown' }, 'font-system'],
  ]) {
    const font = loadSource('utils/system-font.ts', { wx: { getDeviceInfo: () => device } }).exports;
    assert.equal(font.getSystemFontClass(), expected);
  }
  for (const wx of [{}, { getDeviceInfo() { throw new Error('unavailable'); } }]) {
    assert.equal(loadSource('utils/system-font.ts', { wx }).exports.getSystemFontClass(), 'font-system');
  }
  const format = loadSource('utils/format.ts').exports;
  assert.deepEqual([format.formatScore(20), format.formatScore(-20), format.formatScore(0)], ['+20', '−20', '0']);
  const config = JSON.parse(fs.readFileSync(path.join(__dirname, '../src/app.json'), 'utf8'));
  assert.deepEqual(config.tabBar.list.map(tab => tab.text), ['牌局', '战绩', '我的']);
  assert.equal(config.tabBar.custom, true);
  await testCustomTabSelection();
  await testActiveHome();
  await testEmptyAndOffline();
  await testDetailRoutesAndPublicAccess();
  await testCreateAndJoinReturnToTab();
})().catch(error => { console.error(error); process.exitCode = 1; });
