import { requireLogin } from '../../services/auth.service';
import { getRanking } from '../../services/game.service';
import type { RankingItem } from '../../models/game-session';
import { formatScore } from '../../utils/format';

type RankedPlayer = RankingItem & {
  initial: string;
  scoreLabel: string;
  scoreTone: 'positive' | 'negative' | 'muted';
  averageLabel: string;
};

Page({
  data: {
    players: [] as RankedPlayer[]
  },

  async onLoad() {
    wx.showLoading({ title: '加载中...' });
    try {
      await requireLogin();
      const list = await getRanking();
      this.setData({
        players: list.map((item) => ({
          ...item,
          initial: item.displayName.slice(0, 1),
          scoreLabel: formatScore(item.totalScore),
          scoreTone: item.totalScore > 0 ? 'positive' as const : item.totalScore < 0 ? 'negative' as const : 'muted' as const,
          averageLabel: String(item.averageScore || 0),
        })),
      });
    } catch (err) {
      wx.showToast({ title: (err as any).message || '获取排行榜失败', icon: 'none' });
    } finally {
      wx.hideLoading();
    }
  },
});
