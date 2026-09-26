import { instance, loadSource } from '../../helpers/page';

const mockUser = { id: 'user-1', displayName: 'User' };
let mockCurrentRequests = 0;

jest.mock('../../../src/services/auth.service', () => ({
  requireLogin: jest.fn(async () => mockUser),
}));

jest.mock('../../../src/services/game.service', () => ({
  getCurrentGame: jest.fn(async () => {
    mockCurrentRequests++;
    return null;
  }),
  getHistory: jest.fn(async () => ({ items: [] })),
  getHistoryStats: jest.fn(async () => ({ totalGames: 0, maxScore: 0 })),
}));

describe('home page loading', () => {
  beforeEach(() => {
    mockCurrentRequests = 0;
    jest.clearAllMocks();
  });

  it('shares one load across concurrent onShow calls', async () => {
    const wx = { showLoading() {}, hideLoading() {}, showToast() {} };
    const { page } = loadSource('pages/home/index.ts', { wx });
    const home = instance(page!);
    await Promise.all(Array.from({ length: 7 }, () => home.onShow()));
    expect(mockCurrentRequests).toBe(1);
    await home.onShow();
    expect(mockCurrentRequests).toBe(2);
  });
});
