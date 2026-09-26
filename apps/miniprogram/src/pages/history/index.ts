import { requireLogin } from '../../services/auth.service';
import { getHistory } from '../../services/game.service';
import type { HistoryItem } from '../../models/game-session';
import { formatScore, formatFriendlyTime } from '../../utils/format';
import { scoreTone } from '../../utils/performance';
import { getSystemFontClass } from '../../utils/system-font';

type ViewHistoryItem = HistoryItem & {
  meta: string;
  scoreLabel: string;
  scoreTone: string;
};

function buildMeta(item: HistoryItem): string {
  const parts = [
    formatFriendlyTime(item.settledAt),
    `${item.participantCount || 0}人`,
    `${item.scoreTransferCount}笔计分`,
  ];
  if (item.winnerName) {
    parts.push(`胜者: ${item.winnerName} (${formatScore(item.winnerScore || 0)}分)`);
  }
  return parts.filter(Boolean).join(' · ');
}

Page({
  data: {
    fontClass: 'font-system',
    items: [] as ViewHistoryItem[],
    isLoading: false,
    hasMore: true,
    nextCursor: '',
    error: '',
  },

  async onLoad() {
    this.setData({ fontClass: getSystemFontClass() });
    await this.loadData(true);
  },

  async loadData(reload = false) {
    if (this.data.isLoading) return;
    if (!reload && !this.data.hasMore) return;

    this.setData({ isLoading: true });
    wx.showLoading({ title: '加载中...' });
    try {
      await requireLogin();
      const cursor = reload ? '' : this.data.nextCursor;
      const res = await getHistory(cursor, 20);

      const formatted = res.items.map(item => ({
        ...item,
        meta: buildMeta(item),
        scoreLabel: formatScore(item.myFinalScore),
        scoreTone: scoreTone(item.myFinalScore),
      }));

      this.setData({
        items: reload ? formatted : [...this.data.items, ...formatted],
        nextCursor: res.nextCursor || '',
        hasMore: !!res.nextCursor,
        isLoading: false,
        error: '',
      });
    } catch (err) {
      const message = (err as Error).message || '历史牌局加载失败，请检查网络后重试';
      const hasItems = this.data.items.length > 0;
      this.setData({ isLoading: false, error: hasItems ? '' : message });
      if (hasItems) wx.showToast({ title: message, icon: 'none' });
    } finally {
      wx.hideLoading();
    }
  },

  retry() {
    return this.loadData(true);
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
