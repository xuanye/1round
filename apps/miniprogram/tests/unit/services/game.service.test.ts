import { loadSource } from '../../helpers/page';

// http is the only real dependency of game.service; capturing its request
// here replaces the old harness's `require: () => ({ request })` override.
let mockRequestCalls = 0;
let mockLastRequest: Record<string, unknown> | undefined;

jest.mock('../../../src/services/http', () => ({
  request: (value: Record<string, unknown>) => {
    mockRequestCalls++;
    mockLastRequest = value;
    return Promise.resolve({});
  },
  requestBinary: jest.fn(),
}));

describe('game.service', () => {
  beforeEach(() => {
    mockRequestCalls = 0;
    mockLastRequest = undefined;
  });

  it('builds the create-game request body', () => {
    const { exports } = loadSource('services/game.service.ts');
    exports.createGame('朋友局', null, [20, 30, 40, 60]);
    expect(mockLastRequest).toEqual({
      url: '/api/game-sessions',
      method: 'POST',
      data: { name: '朋友局', maxParticipants: null, presetScores: [20, 30, 40, 60] },
    });
  });

  it('rejects a missing game id without reaching HTTP', async () => {
    const { exports } = loadSource('services/game.service.ts');
    await expect(exports.getSummary('')).rejects.toBeDefined();
    expect(mockRequestCalls).toBe(0);
  });
});
