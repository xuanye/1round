import { instance, loadSource } from '../../helpers/page';

const mockUser = { id: 'u1' };
let mockSummary: Record<string, any>;
let mockFail: boolean;
let mockLogins: number;
let mockSockets: Array<Record<string, any>>;

jest.mock('../../../src/services/auth.service', () => ({
  requireLogin: async () => { mockLogins++; return mockUser; },
}));
jest.mock('../../../src/services/game.service', () => ({
  getSummary: async () => {
    if (mockFail) throw new Error('离线');
    return mockSummary;
  },
}));
jest.mock('../../../src/services/realtime.service', () => ({
  RealtimeService: class {
    handler?: () => void;
    disconnected = false;
    constructor() { mockSockets.push(this); }
    connect() {}
    onEvent(handler: () => void) { this.handler = handler; }
    disconnect() { this.disconnected = true; }
  },
}));

function harness() {
  mockFail = false;
  mockLogins = 0;
  mockSockets = [];
  mockSummary = { name: '测试牌局', status: 'active', players: [
    { id: 'p1', userId: 'u1', displayName: 'Alice', totalScore: -20 },
    { id: 'p2', userId: 'u2', displayName: 'Bob', totalScore: 10 },
    { id: 'p3', userId: 'u3', displayName: 'Carol', totalScore: 10 },
  ] };
  const page = instance(loadSource('pages/game-ranking/index.ts').page!);
  page.onLoad({ id: 'game-1' });
  return page;
}

describe('game ranking', () => {
  it('uses authenticated HTTP state and stable score ordering without mutating summary', async () => {
    const page = harness();
    await page.onShow();
    expect(mockLogins).toBe(1);
    expect(page.data.players.map((p: any) => p.id)).toEqual(['p2', 'p3', 'p1']);
    expect(page.data.players[2]).toMatchObject({ isMe: true, scoreLabel: '−20', scoreTone: 'negative' });
    expect(mockSummary.players.map((p: any) => p.id)).toEqual(['p1', 'p2', 'p3']);
    mockSummary.players[0].totalScore = 30;
    mockSockets[0].handler!();
    await new Promise(resolve => setTimeout(resolve, 0));
    expect(page.data.players[0].id).toBe('p1');
    page.onHide();
    expect(mockSockets[0].disconnected).toBe(true);
  });

  it('handles errors, retry, empty results, settlement and invalid links', async () => {
    const page = harness();
    mockFail = true;
    await page.onShow();
    expect(page.data.error).toContain('本局排名加载失败');
    expect(mockSockets).toHaveLength(0);
    mockFail = false;
    mockSummary.players = [];
    await page.loadRanking();
    expect(page.data.error).toBe('');
    expect(page.data.players).toEqual([]);
    expect(mockSockets).toHaveLength(1);
    mockSummary.status = 'finished';
    await page.loadRanking();
    expect(page.data.finished).toBe(true);
    expect(mockSockets[0].disconnected).toBe(true);
    page.onLoad({});
    await page.loadRanking();
    expect(page.data.error).toContain('牌局链接无效');
  });
});
