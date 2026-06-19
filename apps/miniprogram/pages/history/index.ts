import { getRecentSessions } from '../../utils/storage';

Page({
  data: { ids: [] as string[] },
  onShow() {
    this.setData({ ids: getRecentSessions() });
  },
  open(event: WechatMiniprogram.TouchEvent) {
    wx.navigateTo({ url: `/pages/game-detail/index?id=${event.currentTarget.dataset.id}` });
  },
});
