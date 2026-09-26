import { requireLogin } from '../../services/auth.service';

Page({
  data: { displayName: '', initial: '', error: '' },
  async onShow() {
    this.getTabBar?.()?.setData({ selected: 2 });
    try {
      const user = await requireLogin();
      const displayName = user.displayName || '老书记';
      this.setData({ displayName, initial: displayName.slice(0, 1), error: '' });
    } catch (err) {
      this.setData({ error: (err as Error).message || '资料加载失败，请重试' });
    }
  },
  openGame() { wx.switchTab({ url: '/pages/home/index' }); },
  openHistory() { wx.navigateTo({ url: '/pages/history/index' }); },
  showHelp() {
    wx.showModal({
      title: '使用帮助',
      content: '扫码加入或创建牌局后，点击“记一笔”为其他参与者加分，自己扣除对应总分。分值为 0 时可退出牌局。牌局结束后，可在战绩中查看历史。牌局内的展示名称可点击自己的名字修改。',
      showCancel: false,
    });
  },
});
