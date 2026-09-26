// Programmable wx mock for new tests (component tests, new unit tests).
// Migrated legacy tests keep their inline wx objects so assertions stay 1:1.

export interface WxMock {
  showToast: jest.Mock;
  hideLoading: jest.Mock;
  showLoading: jest.Mock;
  navigateTo: jest.Mock;
  navigateBack: jest.Mock;
  switchTab: jest.Mock;
  stopPullDownRefresh: jest.Mock;
  showModal: jest.Mock;
  scanCode: jest.Mock;
  getStorageSync: jest.Mock;
  setStorageSync: jest.Mock;
  getDeviceInfo: jest.Mock;
  createCanvasContext: jest.Mock;
  connectSocket: jest.Mock;
  [key: string]: any;
}

export function createWxMock(overrides: Record<string, unknown> = {}): WxMock {
  return {
    showToast: jest.fn(),
    hideLoading: jest.fn(),
    showLoading: jest.fn(),
    navigateTo: jest.fn(),
    navigateBack: jest.fn(),
    switchTab: jest.fn(),
    stopPullDownRefresh: jest.fn(),
    showModal: jest.fn(),
    scanCode: jest.fn(),
    getStorageSync: jest.fn(() => ''),
    setStorageSync: jest.fn(),
    getDeviceInfo: jest.fn(() => ({ platform: 'devtools', system: 'Unknown' })),
    createCanvasContext: jest.fn(),
    connectSocket: jest.fn(),
    ...overrides,
  };
}
