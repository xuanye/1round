import { instance, loadSource } from '../../helpers/page';

// jest.mock factories are hoisted above these declarations; they only run
// when the mocked module is first required (during loadSource), by which
// point the captures below are initialized.
const mockCalls: unknown[][] = [];
const mockRoutes: string[] = [];
const mockToasts: Array<Record<string, unknown>> = [];
const mockSaved: string[] = [];
let mockCreateGame: (...args: unknown[]) => Promise<{ id: string }> = async () => ({ id: 'created' });

jest.mock('../../../src/services/auth.service', () => ({
  requireLogin: jest.fn(async () => ({ id: 'user' })),
}));

jest.mock('../../../src/services/game.service', () => ({
  createGame: async (...args: unknown[]) => {
    mockCalls.push(args);
    return mockCreateGame(...args);
  },
}));

jest.mock('../../../src/utils/storage', () => ({
  saveRecentSession: (id: string) => { mockSaved.push(id); },
}));

const state = { calls: mockCalls, routes: mockRoutes, toasts: mockToasts, saved: mockSaved };

function setup(createGame: (...args: unknown[]) => Promise<{ id: string }> = async () => ({ id: 'created' })) {
  mockCalls.length = 0;
  mockRoutes.length = 0;
  mockToasts.length = 0;
  mockSaved.length = 0;
  mockCreateGame = createGame;
  const wx = {
    showToast(value: Record<string, unknown>) { mockToasts.push(value); },
    switchTab(value: { url: string }) { mockRoutes.push(value.url); },
  };
  const page = instance(loadSource('pages/game-create/index.ts', { wx }).page!);
  return { page };
}

describe('game-create page', () => {
  it('validates the name and preset scores before creating', async () => {
    const { page } = setup();
    page.selectSuggestedName({ currentTarget: { dataset: { name: '周末牌局' } } });
    expect(page.data.name).toBe('周末牌局');
    expect(page.data.nameLength).toBe(4);
    page.clearName();
    expect(page.data.nameLength).toBe(0);
    expect(page.data.nameFocused).toBe(true);
    await page.submit();
    expect(state.calls).toHaveLength(0);
    expect(state.toasts[0].title).toBe('请输入牌局名称');
    page.onNameInput({ detail: { value: '  家庭聚会  ' } });
    await page.submit();
    expect(state.calls[0]).toEqual(['家庭聚会', null, [20, 30, 40, 60]]);
    page.togglePresetScore({ currentTarget: { dataset: { value: 60 } } });
    await page.submit();
    expect(state.calls[1]).toEqual(['家庭聚会', null, [20, 30, 40]]);
    page.togglePresetScore({ currentTarget: { dataset: { value: 10 } } });
    page.togglePresetScore({ currentTarget: { dataset: { value: 50 } } });
    expect(state.toasts.at(-1)!.title).toBe('最多选 4 个分值');
    await page.submit();
    expect(state.calls[2]).toEqual(['家庭聚会', null, [10, 20, 30, 40]]);
    page.togglePresetScore({ currentTarget: { dataset: { value: 20 } } });
    page.togglePresetScore({ currentTarget: { dataset: { value: 30 } } });
    page.togglePresetScore({ currentTarget: { dataset: { value: 40 } } });
    page.togglePresetScore({ currentTarget: { dataset: { value: 10 } } });
    expect(state.toasts.at(-1)!.title).toBe('至少保留 1 个分值');
    await page.submit();
    expect(state.calls[3]).toEqual(['家庭聚会', null, [10]]);
    expect(state.routes.at(-1)).toBe('/pages/home/index');
    expect(state.saved.at(-1)).toBe('created');
  });

  it('blocks duplicate submits while creation is pending', async () => {
    let resolveRequest!: (value: { id: string }) => void;
    const { page } = setup(() => new Promise((resolve) => { resolveRequest = resolve; }));
    const first = page.submit();
    await Promise.resolve();
    await page.submit();
    expect(state.calls).toHaveLength(1);
    expect(page.data.submitting).toBe(true);
    page.clearName();
    expect(page.data.name).toBe('周六朋友局');
    resolveRequest({ id: 'created' });
    await first;
    expect(page.data.submitting).toBe(false);
  });

  it('surfaces creation errors and allows retry', async () => {
    let fail = true;
    const { page } = setup(async () => {
      if (fail) throw new Error('网络不可用，请重试');
      return { id: 'retry' };
    });
    await page.submit();
    expect(page.data.creationError).toBe('网络不可用，请重试');
    expect(page.data.submitting).toBe(false);
    expect(state.routes).toHaveLength(0);
    fail = false;
    await page.submit();
    expect(page.data.creationError).toBe('');
    expect(state.saved[0]).toBe('retry');
  });
});
