import { instance, loadSource } from '../../helpers/page';

const mockLogin = jest.fn();
jest.mock('../../../src/services/auth.service', () => ({ requireLogin: () => mockLogin() }));

function setup(version = '') {
  const wx = {
    getAccountInfoSync: () => ({ miniProgram: { version } }),
    navigateTo: jest.fn(), showModal: jest.fn(),
  };
  const page = instance(loadSource('pages/mine/index.ts', { wx }).page!);
  const setData = jest.fn();
  page.getTabBar = () => ({ setData });
  return { page, wx, setData };
}

beforeEach(() => mockLogin.mockReset());

describe('mine page', () => {
  it('loads the account identity and selects the mine tab', async () => {
    mockLogin.mockResolvedValue({ displayName: '假正经哥哥', avatarUrl: 'https://example.com/avatar.png' });
    const { page, setData } = setup('1.2.3');
    page.onLoad();
    const loading = page.onShow();
    expect(page.data.loading).toBe(true);
    await loading;
    expect(setData).toHaveBeenCalledWith({ selected: 2 });
    expect(page.data).toMatchObject({ displayName: '假正经哥哥', initial: '假', avatarUrl: 'https://example.com/avatar.png', loading: false, error: '', version: 'v1.2.3' });
    page.onAvatarError();
    expect(page.data.avatarFailed).toBe(true);
    await page.onShow();
    expect(page.data.avatarFailed).toBe(false);
  });

  it('shows a recoverable login error and retries', async () => {
    mockLogin.mockRejectedValueOnce(new Error('网络不可用')).mockResolvedValueOnce({ displayName: null, avatarUrl: null });
    const { page } = setup();
    await page.onShow();
    expect(page.data).toMatchObject({ error: '网络不可用', loading: false });
    await page.onShow();
    expect(page.data).toMatchObject({ error: '', displayName: '老书记', initial: '老', avatarUrl: '', loading: false });
  });

  it('opens existing history and explains help and the actual build version', () => {
    const { page, wx } = setup();
    page.onLoad();
    expect(page.data.version).toBe('开发版');
    page.openHistory();
    expect(wx.navigateTo).toHaveBeenCalledWith({ url: '/pages/history/index' });
    page.showHelp();
    expect(wx.showModal).toHaveBeenLastCalledWith(expect.objectContaining({ title: '使用说明', showCancel: false }));
    page.showAbout();
    expect(wx.showModal).toHaveBeenLastCalledWith(expect.objectContaining({ title: '关于一局一分', content: expect.stringContaining('开发版'), showCancel: false }));
  });
});
