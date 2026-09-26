import { instance, loadSource } from '../../helpers/page';

let mockFail = false;
let mockCursor = '';
jest.mock('../../../src/services/auth.service', () => ({ requireLogin: jest.fn(async () => ({})) }));
jest.mock('../../../src/services/game.service', () => ({
  getHistory: jest.fn(async (cursor: string) => {
    if (mockFail) throw new Error('网络不可用');
    if (cursor) {
      return {
        items: [{ id: 'game-2', name: '跨年局', settledAt: '2026-09-20T12:00:00Z', scoreTransferCount: 6, myFinalScore: -30, participantCount: 3 }],
        nextCursor: '',
      };
    }
    return {
      items: [{
        id: 'game-1', name: '周六朋友局', settledAt: '2026-09-26T02:00:00Z',
        scoreTransferCount: 12, myFinalScore: 45, participantCount: 4,
        winnerName: '阿强', winnerScore: 60,
      }],
      nextCursor: mockCursor,
    };
  }),
}));

function page() {
  return instance(loadSource('pages/history/index.ts', { wx: {
    showLoading: jest.fn(), hideLoading: jest.fn(), showToast: jest.fn(),
    stopPullDownRefresh: jest.fn(), navigateTo: jest.fn(),
  } }).page!);
}

beforeEach(() => { mockFail = false; mockCursor = ''; });

it('formats rows, paginates and opens the game detail', async () => {
  mockCursor = 'cursor-1';
  const history = page();
  await history.onLoad();
  expect(history.data.items).toHaveLength(1);
  expect(history.data.items[0].meta).toContain('4人');
  expect(history.data.items[0].meta).toContain('12笔计分');
  expect(history.data.items[0].meta).toContain('胜者: 阿强 (+60分)');
  expect(history.data.items[0].scoreLabel).toBe('+45');
  expect(history.data.items[0].scoreTone).toBe('positive');
  expect(history.data.hasMore).toBe(true);

  await history.onReachBottom();
  expect(history.data.items).toHaveLength(2);
  expect(history.data.items[1]).toMatchObject({ scoreLabel: '−30', scoreTone: 'negative' });
  expect(history.data.hasMore).toBe(false);

  history.open({ currentTarget: { dataset: { id: 'game-1' } } });
  expect(wx.navigateTo).toHaveBeenCalledWith({ url: '/pages/game-detail/index?id=game-1' });
});

it('shows a recoverable error when the first page fails and clears it on retry', async () => {
  mockFail = true;
  const history = page();
  await history.onLoad();
  expect(history.data).toMatchObject({ error: '网络不可用', isLoading: false, items: [] });

  mockFail = false;
  await history.retry();
  expect(history.data.error).toBe('');
  expect(history.data.items).toHaveLength(1);
});
