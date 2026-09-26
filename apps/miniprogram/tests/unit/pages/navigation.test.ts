import { instance, loadSource } from '../../helpers/page';

const fs = require('node:fs');
const path = require('node:path');

// jest.mock factories are hoisted; they read `mockState`/`mockUser` lazily on
// first require, after these module-level bindings are initialized. Each
// harness() call swaps in a fresh state object for the next test.
const mockUser = { id: 'user-1', displayName: 'Alice' };

interface NavState {
  current: { id: string; inviteCode: string } | null;
  score: number;
  summaryCalls: number;
  historyCalls: number;
  logins: number;
  sockets: Array<Record<string, any>>;
  routes: unknown[][];
  failCurrent: boolean;
  modal?: { success: (value: { confirm: boolean }) => Promise<void> } & Record<string, any>;
}

let mockState: NavState;

jest.mock('../../../src/services/auth.service', () => ({
  requireLogin: jest.fn(async () => {
    mockState.logins++;
    return mockUser;
  }),
}));

jest.mock('../../../src/services/game.service', () => ({
  getCurrentGame: jest.fn(async () => {
    if (mockState.failCurrent) throw new Error('离线');
    return mockState.current;
  }),
  getSummary: jest.fn(async () => {
    mockState.summaryCalls++;
    return {
      name: 'Test game',
      status: 'active',
      ownerUserId: mockUser.id,
      inviteCode: 'ABC123',
      updatedAt: '2026-09-26T00:00:00Z',
      players: [
        { id: 'p1', userId: mockUser.id, displayName: 'Alice', totalScore: mockState.score },
        { id: 'p2', userId: 'user-2', displayName: 'Bob', totalScore: -mockState.score },
      ],
      roundStatus: null,
    };
  }),
  getScoreTransfers: jest.fn(async () => []),
  getHistory: jest.fn(async () => {
    mockState.historyCalls++;
    return { items: [] };
  }),
  leaveGame: jest.fn(async () => {
    mockState.current = null;
  }),
  createGame: jest.fn(async () => ({ id: 'new-game', inviteCode: 'ABC123' })),
  joinGame: jest.fn(async () => ({ gameSessionId: 'new-game' })),
}));

jest.mock('../../../src/utils/storage', () => ({
  getUser: () => mockUser,
  saveRecentSession: () => {},
}));

jest.mock('../../../src/services/realtime.service', () => {
  class RealtimeService {
    id?: string;
    handler?: () => void;
    disconnected?: boolean;
    constructor() {
      mockState.sockets.push(this as unknown as Record<string, any>);
    }
    connect(id: string) { this.id = id; }
    onEvent(handler: () => void) { this.handler = handler; }
    disconnect() { this.disconnected = true; }
  }
  return { RealtimeService };
});

function newState(): NavState {
  return {
    current: { id: 'game-1', inviteCode: 'ABC123' },
    score: 0,
    summaryCalls: 0,
    historyCalls: 0,
    logins: 0,
    sockets: [],
    routes: [],
    failCurrent: false,
    modal: undefined,
  };
}

function harness() {
  mockState = newState();
  const wx = {
    showLoading() {},
    hideLoading() {},
    showToast() {},
    stopPullDownRefresh() {},
    switchTab({ url }: { url: string }) { mockState.routes.push(['tab', url]); },
    navigateTo({ url }: { url: string }) { mockState.routes.push(['page', url]); },
    showModal(options: { success: (value: { confirm: boolean }) => Promise<void> }) {
      (mockState as NavState).modal = options;
    },
  };
  const home = instance(loadSource('pages/home/index.ts', { wx }).page!);
  return { state: mockState, home, wx };
}

describe('navigation', () => {
  it('derives system font classes from device info', () => {
    for (const [device, expected] of [
      [{ platform: 'ios', system: 'iOS 18' }, 'font-ios'],
      [{ platform: 'android', system: 'Android 15', brand: 'HUAWEI' }, 'font-android'],
      [{ platform: 'ohos', system: 'HarmonyOS 5' }, 'font-harmony'],
      [{ platform: 'android', system: 'HarmonyOS 4' }, 'font-harmony'],
      [{ platform: 'devtools', system: 'Unknown' }, 'font-system'],
    ] as Array<[{ platform: string; system: string; brand?: string }, string]>) {
      const font = loadSource('utils/system-font.ts', { wx: { getDeviceInfo: () => device } }).exports;
      expect(font.getSystemFontClass()).toBe(expected);
    }
    for (const wx of [{}, { getDeviceInfo() { throw new Error('unavailable'); } }]) {
      expect(loadSource('utils/system-font.ts', { wx }).exports.getSystemFontClass()).toBe('font-system');
    }
  });

  it('formats scores with unicode minus', () => {
    const format = loadSource('utils/format.ts').exports;
    expect([format.formatScore(20), format.formatScore(-20), format.formatScore(0)]).toEqual(['+20', '−20', '0']);
  });

  it('declares a custom tab bar with the expected tabs', () => {
    const config = JSON.parse(fs.readFileSync(path.join(__dirname, '../../../src/app.json'), 'utf8'));
    expect(config.tabBar.list.map((tab: { text: string }) => tab.text)).toEqual(['牌局', '战绩', '我的']);
    expect(config.tabBar.custom).toBe(true);
  });

  it('restores tab selection on show and routes tab-bar taps', async () => {
    const { state, home, wx } = harness();
    const selected: number[] = [];
    home.getTabBar = () => ({ setData: (data: { selected: number }) => selected.push(data.selected) });
    await home.onShow();
    expect(selected.pop()).toBe(0);

    const mine = instance(loadSource('pages/mine/index.ts', { wx }).page!);
    mine.getTabBar = () => ({ setData: (data: { selected: number }) => selected.push(data.selected) });
    await mine.onShow();
    expect(selected.pop()).toBe(2);

    const { component } = loadSource('custom-tab-bar/index.ts', { wx });
    expect(component).toBeDefined();
    const tabBar = { data: component!.data };
    const select = (index: number) =>
      component!.methods.selectTab.call(tabBar, { currentTarget: { dataset: { index } } });
    select(0);
    select(99);
    expect(state.routes).toHaveLength(0);
    select(1);
    expect(state.routes.pop()).toEqual(['tab', '/pages/ranking/index']);
    select(2);
    expect(state.routes.pop()).toEqual(['tab', '/pages/mine/index']);
  });

  it('renders the active home, refreshes on events, and guards exit', async () => {
    const { state, home } = harness();
    await home.onShow();
    expect(home.data.homeState).toBe('ready');
    expect(home.data.id).toBe('game-1');
    expect(home.data.participants).toHaveLength(2);
    expect(state.historyCalls).toBe(0);
    home.inputScore();
    expect(state.routes.pop()).toEqual(['page', '/pages/score-input/index?id=game-1']);
    state.score = 20;
    await state.sockets[0].handler();
    // The event handler schedules an HTTP refresh; allow its awaited
    // operations to finish.
    await new Promise((resolve) => setTimeout(resolve, 0));
    expect(home.data.participants[0].score).toBe('+20');
    home.onHide();
    expect(state.sockets[0].disconnected).toBe(true);
    await home.onShow();
    expect(state.sockets).toHaveLength(2);
    home.exitGame();
    expect(state.modal).toBeUndefined();
    state.score = 0;
    await home.refreshHome();
    home.exitGame();
    await state.modal!.success({ confirm: true });
    expect(home.data.id).toBe('');
    expect(home.data.homeState).toBe('ready');
    expect(state.historyCalls).toBe(1);
  });

  it('handles empty and offline home states', async () => {
    const { state, home } = harness();
    state.current = null;
    await home.onShow();
    expect(home.data.id).toBe('');
    expect(state.summaryCalls).toBe(0);
    home.createGame();
    expect(state.routes.pop()).toEqual(['page', '/pages/game-create/index']);
    state.failCurrent = true;
    await home.onShow();
    expect(home.data.homeState).toBe('error');
    expect(home.data.homeError).toBe('离线');
    state.failCurrent = false;
    await home.refreshHome();
    expect(home.data.homeState).toBe('ready');
  });

  it('routes game-detail and keeps the public settlement path login-free', async () => {
    const { state } = harness();
    const page = instance(loadSource('pages/game-detail/index.ts', { wx: harnessWx() }).page!);
    await page.onLoad({ id: 'game-1' });
    await page.onShow();
    expect(state.routes.pop()).toEqual(['tab', '/pages/home/index']);

    const publicPage = instance(loadSource('pages/game-detail/index.ts', { wx: harnessWx() }).page!);
    let publicLoads = 0;
    publicPage.loadPublicSettlement = async () => { publicLoads++; };
    await publicPage.onLoad({ shareToken: 'public-token' });
    const loginsBefore = state.logins;
    await publicPage.onShow();
    expect(publicLoads).toBe(1);
    expect(state.logins).toBe(loginsBefore);
  });

  it('returns create/join flows to the game tab', async () => {
    for (const file of ['game-create', 'game-join']) {
      let destination: string | undefined;
      const wx = {
        showLoading() {},
        hideLoading() {},
        showToast() {},
        switchTab({ url }: { url: string }) { destination = url; },
      };
      const page = instance(loadSource(`pages/${file}/index.ts`, { wx }).page!);
      page.data.displayName = 'Alice';
      await page.submit();
      expect(destination).toBe('/pages/home/index');
    }
  });
});

// wx object for detail-page loads; routes land in the current harness state,
// mirroring how the old harness shared one globals bag per harness().
function harnessWx() {
  return {
    showLoading() {},
    hideLoading() {},
    showToast() {},
    stopPullDownRefresh() {},
    switchTab({ url }: { url: string }) { mockState.routes.push(['tab', url]); },
    navigateTo({ url }: { url: string }) { mockState.routes.push(['page', url]); },
    showModal(options: { success: (value: { confirm: boolean }) => Promise<void> }) {
      mockState.modal = options;
    },
  };
}
