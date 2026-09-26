const assert = require('node:assert/strict');
const { loadSource, instance } = require('./page-test-utils');

(async () => {
  for (const presetScores of [[25, 35, 45, 65], undefined]) {
    const page = instance(loadSource('pages/score-input/index.ts', {
      wx: { showLoading() {}, hideLoading() {}, showToast() {} },
      require(name) {
        if (name.includes('auth.service')) return { requireLogin: async () => ({ id: 'owner' }) };
        if (name.includes('game.service')) return { getSummary: async () => ({ presetScores, players: [
          { id: 'p1', userId: 'owner', displayName: '自己', totalScore: 0 },
          { id: 'p2', userId: 'friend', displayName: '牌友', totalScore: 0 },
        ] }) };
        return { getUser: () => ({ id: 'owner', displayName: '自己' }) };
      },
    }).page);
    await page.onLoad({ id: 'game' });
    const expected = presetScores || [20, 30, 40, 60];
    assert.deepEqual(Array.from(page.data.presetScores), expected);
    assert.equal(page.data.scoreText, String(expected[0]));
    assert.equal(page.data.canSubmit, false);
    page.toggleReceiver({ currentTarget: { dataset: { id: 'p2' } } });
    page.quickScore({ currentTarget: { dataset: { value: expected[3] } } });
    assert.equal(page.data.submitText, `给 牌友 +${expected[3]}`);
    page.pressKey({ currentTarget: { dataset: { value: '1' } } });
    assert.equal(page.data.scoreText, `${expected[3]}1`);
  }
  let request;
  const service = loadSource('services/game.service.ts', { require: () => ({ request: value => { request = value; } }) }).exports;
  service.createGame('朋友局', null, [20, 30, 40, 60]);
  assert.deepEqual(JSON.parse(JSON.stringify(request)), {
    url: '/api/game-sessions', method: 'POST', data: { name: '朋友局', maxParticipants: null, presetScores: [20, 30, 40, 60] },
  });
})().catch(error => { console.error(error); process.exitCode = 1; });
