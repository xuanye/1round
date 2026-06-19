import { ensureLogin } from '../../services/auth.service';
import { getHistory } from '../../services/game.service';
import type { HistoryItem } from '../../models/game-session';
import { formatScore } from '../../utils/format';

type ViewHistoryItem = HistoryItem & {
  dateText: string;
  scoreText: string;
};

Page({
  data: {
    items: [] as ViewHistoryItem[],
    isLoading: false,
    hasMore: true,
    nextCursor: '',
  },

  async onLoad() {
    await this.loadData(true);
  },

  async loadData(reload = false) {
    if (this.data.isLoading) return;
    if (!reload && !this.data.hasMore) return;

    this.setData({ isLoading: true });
    wx.showLoading({ title: '加载中...' });
    try {
      await ensureLogin();
      const cursor = reload ? '' : this.data.nextCursor;
      const res = await getHistory(cursor, 20);

      const formatted = res.items.map(item => {
        const dateText = new Date(item.settledAt).toLocaleDateString('zh-CN', {
          month: '2-digit',
          day: '2-digit',
          hour: '2-digit',
          minute: '2-digit',
        });
        return {
          ...item,
          dateText,
          scoreText: formatScore(item.myFinalScore),
        };
      });

      const nextItems = reload ? formatted : [...this.data.items, ...formatted];
      this.setData({
        items: nextItems,
        nextCursor: res.nextCursor || '',
        hasMore: !!res.nextCursor,
        isLoading: false,
      });
    } catch (err) {
      this.setData({ isLoading: false });
      wx.showToast({ title: (err as any).message || '加载历史失败', icon: 'none' });
    } finally {
      wx.hideLoading();
    }
  },

  async onReachBottom() {
    await this.loadData();
  },

  async onPullDownRefresh() {
    await this.loadData(true);
    wx.stopPullDownRefresh();
  },

  open(event: WechatMiniprogram.TouchEvent) {
    const id = String(event.currentTarget.dataset.id);
    wx.navigateTo({ url: `/pages/game-detail/index?id=${id}` });
  },
});
