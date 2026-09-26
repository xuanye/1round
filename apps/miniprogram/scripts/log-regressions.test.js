const assert = require('assert');

const { loadSource } = require('./page-test-utils');

async function testReconnectRetainsNotifications() {
  const sockets = [];
  let reconnect;
  const wx = {
    connectSocket() {
      const socket = {
        onMessage(callback) { this.message = callback; },
        onClose(callback) { this.closed = callback; },
        onError(callback) { this.error = callback; },
        close() {},
      };
      sockets.push(socket);
      return socket;
    },
  };
  const { exports } = loadSource('services/realtime.service.ts', {
    wx,
    getApp: () => ({ globalData: { baseUrl: 'https://example.test' } }),
    require: () => ({ getToken: () => 'test-token' }),
    setTimeout: (callback) => { reconnect = callback; return 1; },
    clearTimeout() {},
  });
  const service = new exports.RealtimeService();
  let notifications = 0;
  service.connect('game-1');
  service.onEvent(() => { notifications++; });
  sockets[0].message({ data: '{"type":"score_transfer.submitted"}' });
  sockets[0].closed();
  reconnect();
  sockets[1].message({ data: '{"type":"score_transfer.submitted"}' });
  assert.strictEqual(notifications, 2, 'notifications must still reach the page after reconnect');
}

async function testHomeLoadsOnceWhenShownConcurrently() {
  let currentRequests = 0;
  const { page } = loadSource('pages/home/index.ts', {
    wx: { showLoading() {}, hideLoading() {}, showToast() {} },
    require(name) {
      if (name.includes('auth.service')) return { requireLogin: async () => ({ id: 'user-1', displayName: 'User' }) };
      if (name.includes('game.service')) return {
        getCurrentGame: async () => { currentRequests++; return null; },
        getHistory: async () => ({ items: [] }),
        getHistoryStats: async () => ({ totalGames: 0, maxScore: 0 }),
      };
      return {};
    },
  });
  const instance = { ...page, data: { ...page.data }, setData(next) { Object.assign(this.data, next); } };
  await Promise.all(Array.from({ length: 7 }, () => instance.onShow()));
  assert.strictEqual(currentRequests, 1, 'concurrent onShow calls must share one load');
  await instance.onShow();
  assert.strictEqual(currentRequests, 2, 'a later onShow must refresh again');
}

async function testScorePageRejectsMissingGameID() {
  let summaryRequests = 0;
  let redirectedTo = '';
  const { page } = loadSource('pages/score-input/index.ts', {
    wx: {
      showToast() {},
      showLoading() {},
      hideLoading() {},
      switchTab({ url }) { redirectedTo = url; },
    },
    setTimeout: (callback) => callback(),
    require(name) {
      if (name.includes('auth.service')) return { requireLogin: async () => ({ id: 'user-1' }) };
      if (name.includes('game.service')) return { getSummary: async () => { summaryRequests++; return { players: [] }; } };
      if (name.includes('storage')) return { getUser: () => ({ id: 'user-1' }) };
      return {};
    },
  });
  const instance = { ...page, data: { ...page.data }, setData(next) { Object.assign(this.data, next); } };
  await instance.onLoad({});
  assert.strictEqual(summaryRequests, 0, 'missing ID must not call summary');
  assert.strictEqual(redirectedTo, '/pages/home/index');
}

async function testSummaryServiceRejectsMissingGameID() {
  let requests = 0;
  const { exports } = loadSource('services/game.service.ts', {
    require: () => ({ request: () => { requests++; } }),
  });
  await assert.rejects(exports.getSummary(''));
  assert.strictEqual(requests, 0, 'missing ID must not reach the HTTP service');
}

function testSettlementSceneParsing() {
  const { page } = loadSource('pages/game-detail/index.ts', {
    require: () => ({}),
  });
  for (const [scene, expected] of [
    ['Abcdef0123456789_-Abcdef', 'Abcdef0123456789_-Abcdef'],
    ['036662a691264225bc19ab55e1748311', '036662a6-9126-4225-bc19-ab55e1748311'],
    ['shareToken=Abcdef0123456789_-Abcdef', 'Abcdef0123456789_-Abcdef'],
  ]) {
    const instance = { ...page, data: { ...page.data }, setData(next) { Object.assign(this.data, next); } };
    instance.onLoad({ scene });
    assert.strictEqual(instance.data.shareToken, expected, `scene ${scene}`);
  }
}

(async () => {
  await testReconnectRetainsNotifications();
  await testHomeLoadsOnceWhenShownConcurrently();
  await testScorePageRejectsMissingGameID();
  await testSummaryServiceRejectsMissingGameID();
  testSettlementSceneParsing();
})().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
