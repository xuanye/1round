import { instance, loadSource } from '../../helpers/page';

const mockGetSummary = jest.fn();
const mockRequireLogin = jest.fn(async () => ({ id: 'owner' }));
const mockGetUser = jest.fn(() => ({ id: 'owner', displayName: '自己' }));

jest.mock('../../../src/services/auth.service', () => ({
  requireLogin: () => mockRequireLogin(),
}));

jest.mock('../../../src/services/game.service', () => ({
  getSummary: (id: string) => mockGetSummary(id),
}));

jest.mock('../../../src/utils/storage', () => ({
  getUser: () => mockGetUser(),
}));

describe('score-input page', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockRequireLogin.mockResolvedValue({ id: 'owner' });
    mockGetUser.mockReturnValue({ id: 'owner', displayName: '自己' });
  });

  it('applies preset scores from the summary', async () => {
    for (const presetScores of [[25, 35, 45, 65], undefined]) {
      mockGetSummary.mockResolvedValueOnce({
        presetScores,
        players: [
          { id: 'p1', userId: 'owner', displayName: '自己', totalScore: 0 },
          { id: 'p2', userId: 'friend', displayName: '牌友', totalScore: 0 },
        ],
      });
      const wx = { showLoading() {}, hideLoading() {}, showToast() {} };
      const page = instance(loadSource('pages/score-input/index.ts', { wx }).page!);
      await page.onLoad({ id: 'game' });
      const expected = presetScores || [20, 30, 40, 60];
      expect(Array.from(page.data.presetScores)).toEqual(expected);
      expect(page.data.scoreText).toBe(String(expected[0]));
      expect(page.data.canSubmit).toBe(false);
      page.toggleReceiver({ currentTarget: { dataset: { id: 'p2' } } });
      page.quickScore({ currentTarget: { dataset: { value: expected[3] } } });
      expect(page.data.submitText).toBe(`给 牌友 +${expected[3]}`);
      page.pressKey({ currentTarget: { dataset: { value: '1' } } });
      expect(page.data.scoreText).toBe(`${expected[3]}1`);
    }
  });

  it('rejects a missing game id without loading the summary', async () => {
    let redirectedTo = '';
    const wx = {
      showToast() {},
      showLoading() {},
      hideLoading() {},
      switchTab({ url }: { url: string }) { redirectedTo = url; },
    };
    const page = instance(loadSource('pages/score-input/index.ts', { wx }).page!);
    jest.useFakeTimers();
    try {
      await page.onLoad({});
      jest.runOnlyPendingTimers();
      expect(mockGetSummary).not.toHaveBeenCalled();
      expect(redirectedTo).toBe('/pages/home/index');
    } finally {
      jest.useRealTimers();
    }
  });
});
