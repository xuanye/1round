import { requireLogin } from '../../services/auth.service';
import { getHistory, getRanking } from '../../services/game.service';
import type { HistoryItem, RankingItem } from '../../models/game-session';
import { formatFriendlyTime, formatScore } from '../../utils/format';

type RankedPlayer = RankingItem & {
  initial: string;
  scoreLabel: string;
  scoreTone: 'positive' | 'negative' | 'muted';
  averageLabel: string;
};

type HistoryPreviewItem = HistoryItem & {
  accentTone: 'green' | 'yellow';
  meta: string;
};

Page({
  data: {
    players: [] as RankedPlayer[],
    historyItems: [] as HistoryPreviewItem[],
  },

  async onShow() {
    wx.showLoading({ title: '加载中...' });
    try {
      await requireLogin();
      const [list, historyPage] = await Promise.all([
        getRanking(),
        getHistory('', 5),
      ]);

      this.setData({
        players: list.map((item) => ({
          ...item,
          initial: item.displayName.slice(0, 1),
          scoreLabel: formatScore(item.totalScore),
          scoreTone: item.totalScore > 0 ? 'positive' as const : item.totalScore < 0 ? 'negative' as const : 'muted' as const,
          averageLabel: String(item.averageScore || 0),
        })),
        historyItems: historyPage.items.map((item, index) => ({
          ...item,
          accentTone: index % 2 === 0 ? 'green' : 'yellow',
          meta: this.buildHistoryMeta(item),
        })),
      });
    } catch (err) {
      wx.showToast({ title: (err as any).message || '获取排行榜失败', icon: 'none' });
    } finally {
      wx.hideLoading();
    }
  },

  buildHistoryMeta(item: HistoryItem) {
    const parts = [
      formatFriendlyTime(item.settledAt),
      `${item.participantCount || 0}人`,
      `${item.scoreTransferCount}局`,
    ];
    if (item.winnerName) {
      const winnerScore = item.winnerScore ?? 0;
      parts.push(`胜者: ${item.winnerName} (${winnerScore > 0 ? '+' : ''}${winnerScore}分)`);
    }
    return parts.join(' · ');
  },

  openHistory() {
    wx.navigateTo({ url: '/pages/history/index' });
  },

  openDetail(event: WechatMiniprogram.TouchEvent) {
    const id = String(event.currentTarget.dataset.id || '');
    if (!id) return;
    wx.navigateTo({ url: `/pages/game-detail/index?id=${id}` });
  },
});
