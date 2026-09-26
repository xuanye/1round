import { instance, loadSource } from '../../helpers/page';
import { chartGeometry, performanceRange } from '../../../src/utils/performance';

let mockFail = false;
let mockCalls = 0;
jest.mock('../../../src/services/auth.service', () => ({ requireLogin: jest.fn(async () => ({})) }));
jest.mock('../../../src/services/game.service', () => ({
  getPerformance: jest.fn(async () => {
    mockCalls++;
    if (mockFail) throw new Error('网络不可用');
    return {
      totalScore: 460, totalGames: 32, wins: 18, maxScore: 120,
      trend: [{ settledAt: '2026-09-25T12:00:00Z', score: 460 }],
      players: [860, 460, -120, 0].map((totalScore, i) => ({ id: String(i), displayName: `牌友${i}`, avatarUrl: '', totalScore, isMe: i === 1 })),
      recentGames: [{ id: 'game-1', name: '周六朋友局', settledAt: '2026-09-25T12:00:00Z', participantCount: 4, myFinalScore: 40 }],
    };
  }),
}));

function page() {
  return instance(loadSource('pages/ranking/index.ts', { wx: {
    nextTick() {}, navigateTo: jest.fn(), stopPullDownRefresh: jest.fn(),
  } }).page!);
}

beforeEach(() => { mockFail = false; mockCalls = 0; });
it('loads once, formats real scores and expands the peer list', async () => {
  const ranking = page();
  await Promise.all([ranking.loadPerformance(), ranking.loadPerformance()]);
  expect(mockCalls).toBe(1);
  expect(ranking.data.totalScoreLabel).toBe('+460');
  expect(ranking.data.visiblePlayers).toHaveLength(3);
  expect(ranking.data.players[2].scoreLabel).toBe('−120');
  expect(ranking.data.historyItems[0].meta).toContain('4人');
  ranking.togglePlayers();
  expect(ranking.data.visiblePlayers).toHaveLength(4);
  ranking.openDetail({ currentTarget: { dataset: { id: 'game-1' } } });
  expect(wx.navigateTo).toHaveBeenCalledWith({ url: '/pages/game-detail/index?id=game-1' });
});
it('shows a recoverable error and clears it on retry', async () => {
  const ranking = page(); mockFail = true;
  await ranking.loadPerformance();
  expect(ranking.data.error).toBe('战绩加载失败，请检查网络后重试');
  expect(ranking.data.loading).toBe(false);
  mockFail = false; await ranking.loadPerformance();
  expect(ranking.data.error).toBe('');
});
it('uses calendar month boundaries and an exclusive end after today', () => {
  const range = performanceRange(6, new Date(2026, 8, 26));
  expect(range.label).toBe('2026.04.01 — 2026.09.26');
  expect(new Date(range.end).getDate()).toBe(27);
});
it('keeps negative, zero and sparse chart values within the plot', () => {
  const start = '2026-04-01T00:00:00Z', end = '2026-10-01T00:00:00Z';
  const chart = chartGeometry([{ settledAt: '2026-04-03T00:00:00Z', score: -120 }, { settledAt: '2026-09-25T00:00:00Z', score: 460 }], start, end, 280, 170);
  for (const point of chart.points) {
    expect(point.x).toBeGreaterThanOrEqual(chart.left);
    expect(point.x).toBeLessThanOrEqual(chart.right);
    expect(point.y).toBeGreaterThanOrEqual(chart.yTop);
    expect(point.y).toBeLessThanOrEqual(chart.yBottom);
  }
  expect(chart.labels).toHaveLength(6);
  expect(chartGeometry([], start, end, 280, 170).scoreLabel).toBe('0');
});
it('changes the period and reloads the selected range', async () => {
  const ranking = page();
  await ranking.loadPerformance();
  ranking.changePeriod({ detail: { value: '1' } });
  await Promise.resolve(); await Promise.resolve();
  expect(ranking.data.periodIndex).toBe(1);
  expect(mockCalls).toBe(2);
  expect(ranking.data.loading).toBe(false);
});
