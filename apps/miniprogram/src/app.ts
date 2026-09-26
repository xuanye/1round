import { getBaseUrl } from './utils/config';

App({
  onLaunch() {
    // Local subset Material Symbols font is loaded offline in app.wxss
  },
  onError(err: string) {
    console.error('Global application error caught:', err);
  },
  onPageNotFound(res: any) {
    console.error('Page not found:', res.path);
    wx.switchTab({ url: '/pages/home/index' });
  },
  globalData: {
    baseUrl: getBaseUrl(),
  },
});

