const assert = require('node:assert/strict');
const { loadSource, instance } = require('./page-test-utils');

function setup(createGame = async () => ({ id: 'created' })) {
  const state = { calls: [], routes: [], toasts: [], saved: [] };
  const page = instance(loadSource('pages/game-create/index.ts', {
    wx: {
      showToast(value) { state.toasts.push(value); },
      switchTab(value) { state.routes.push(value.url); },
    },
    require(name) {
      if (name.includes('auth.service')) return { requireLogin: async () => ({ id: 'user' }) };
      if (name.includes('game.service')) return { createGame: async (...args) => {
        state.calls.push(args);
        return createGame(...args);
      } };
      return { saveRecentSession(id) { state.saved.push(id); } };
    },
  }).page);
  return { page, state };
}

(async () => {
  const { page, state } = setup();
  page.selectSuggestedName({ currentTarget: { dataset: { name: '周末牌局' } } });
  assert.equal(page.data.name, '周末牌局');
  assert.equal(page.data.nameLength, 4);
  page.clearName();
  assert.equal(page.data.nameLength, 0);
  assert.equal(page.data.nameFocused, true);
  await page.submit();
  assert.equal(state.calls.length, 0);
  assert.equal(state.toasts[0].title, '请输入牌局名称');
  page.onNameInput({ detail: { value: '  家庭聚会  ' } });
  await page.submit();
  assert.deepEqual(JSON.parse(JSON.stringify(state.calls[0])), ['家庭聚会', null, [20, 30, 40, 60]]);
  page.togglePresetScore({ currentTarget: { dataset: { value: 60 } } });
  await page.submit();
  assert.deepEqual(JSON.parse(JSON.stringify(state.calls[1])), ['家庭聚会', null, [20, 30, 40]]);
  page.togglePresetScore({ currentTarget: { dataset: { value: 10 } } });
  page.togglePresetScore({ currentTarget: { dataset: { value: 50 } } });
  assert.equal(state.toasts.at(-1).title, '最多选 4 个分值');
  await page.submit();
  assert.deepEqual(JSON.parse(JSON.stringify(state.calls[2])), ['家庭聚会', null, [10, 20, 30, 40]]);
  page.togglePresetScore({ currentTarget: { dataset: { value: 20 } } });
  page.togglePresetScore({ currentTarget: { dataset: { value: 30 } } });
  page.togglePresetScore({ currentTarget: { dataset: { value: 40 } } });
  page.togglePresetScore({ currentTarget: { dataset: { value: 10 } } });
  assert.equal(state.toasts.at(-1).title, '至少保留 1 个分值');
  await page.submit();
  assert.deepEqual(JSON.parse(JSON.stringify(state.calls[3])), ['家庭聚会', null, [10]]);
  assert.equal(state.routes.at(-1), '/pages/home/index');
  assert.equal(state.saved.at(-1), 'created');

  let resolveRequest;
  const pending = setup(() => new Promise(resolve => { resolveRequest = resolve; }));
  const first = pending.page.submit();
  await Promise.resolve();
  await pending.page.submit();
  assert.equal(pending.state.calls.length, 1);
  assert.equal(pending.page.data.submitting, true);
  pending.page.clearName();
  assert.equal(pending.page.data.name, '周六朋友局');
  resolveRequest({ id: 'created' });
  await first;
  assert.equal(pending.page.data.submitting, false);

  let fail = true;
  const retry = setup(async () => {
    if (fail) throw new Error('网络不可用，请重试');
    return { id: 'retry' };
  });
  await retry.page.submit();
  assert.equal(retry.page.data.creationError, '网络不可用，请重试');
  assert.equal(retry.page.data.submitting, false);
  assert.equal(retry.state.routes.length, 0);
  fail = false;
  await retry.page.submit();
  assert.equal(retry.page.data.creationError, '');
  assert.equal(retry.state.saved[0], 'retry');
})().catch(error => { console.error(error); process.exitCode = 1; });
