import { requireLogin } from '../../services/auth.service';
import { getCurrentGame, getSummary, getHistory, getHistoryStats, leaveGame } from '../../services/game.service';
import { formatFriendlyTime } from '../../utils/format';

type HomeCurrentGame = {
  id: string;
  name: string;
  inviteCode: string;
  participantCount: number;
  myScore: number;
  canExit: boolean;
};

type RecentHomeGame = {
  id: string;
  title: string;
  meta: string;
  status: string;
  winnerName: string;
  winnerScoreText: string;
  iconCode: string;
};

Page({
  data: {
    icons: {
      dice: '\uf522',
      enter: '\uf2f6',
      exit: '\uf2f5',
      plusCircle: '\uf055',
      history: '\uf1da',
      home: '\uf015',
      ranking: '\ue561',
    },
    userName: '',
    currentGame: null as HomeCurrentGame | null,
    stats: [
      { label: '聚会次数', value: '0', unit: '场', tone: 'primary', iconCode: '\uf06b' },
      { label: '最高得分', value: '0', unit: '', tone: 'tertiary', iconCode: '\uf5a2' },
    ],
    recentGames: [] as RecentHomeGame[],
  },

  async onShow() {
    wx.showLoading({ title: '加载中...' });
    try {
      const user = await requireLogin();
      this.setData({ userName: user.displayName || '老书记' });

      // Fetch current game
      const current = await getCurrentGame();
      let homeCurrent: HomeCurrentGame | null = null;
      if (current?.id) {
        const summary = await getSummary(current.id);
        // Find my score using robust userId matching
        let myScore = 0;
        const myPlayer = summary.players.find(p => p.userId === user?.id || (p.displayName === user?.displayName && p.userId === user?.id));
        if (myPlayer) {
          myScore = myPlayer.totalScore;
        } else {
          // Fallback matching by name
          const matched = summary.players.find(p => p.displayName === user?.displayName);
          if (matched) myScore = matched.totalScore;
        }

        homeCurrent = {
          id: current.id,
          name: current.name,
          inviteCode: current.inviteCode,
          participantCount: summary.players.length,
          myScore: myScore,
          canExit: myScore === 0,
        };
      } else if (current) {
        console.warn('Ignoring malformed current game response:', current);
      }
      this.setData({ currentGame: homeCurrent });

      // Fetch history
      const historyPage = await getHistory('', 5);
      const recent = historyPage.items.map(item => {
        const friendlyTime = formatFriendlyTime(item.settledAt);
        const settledDate = new Date(item.settledAt);
        const now = new Date();
        const isOver24h = now.getTime() - settledDate.getTime() > 24 * 3600 * 1000;
        let status = '已结束';
        if (isOver24h && item.createdAt) {
          const startDate = new Date(item.createdAt);
          status = `${String(startDate.getMonth() + 1).padStart(2, '0')}/${String(startDate.getDate()).padStart(2, '0')}`;
        }
        return {
          id: item.id,
          title: item.name,
          meta: `${friendlyTime} · ${item.participantCount || 0}人 · ${item.scoreTransferCount}局`,
          status,
          winnerName: item.winnerName || '',
          winnerScoreText: item.winnerScore !== undefined ? `${item.winnerScore > 0 ? '+' : ''}${item.winnerScore}` : '',
          iconCode: '\uf522',
        };
      });
      this.setData({ recentGames: recent });

      // Fetch accurate global statistics from stats API
      const statsData = await getHistoryStats();
      this.setData({
        stats: [
          { label: '聚会次数', value: String(statsData.totalGames), unit: '场', tone: 'primary', iconCode: '\uf06b' },
          { label: '最高得分', value: `${statsData.maxScore >= 0 ? '+' : ''}${statsData.maxScore}`, unit: '', tone: 'tertiary', iconCode: '\uf5a2' },
        ],
      });
    } catch (err) {
      console.error('Home page load failed:', err);
      wx.showToast({ title: (err as any).message || '加载失败', icon: 'none' });
    } finally {
      wx.hideLoading();
    }
  },

  createGame() {
    wx.navigateTo({ url: '/pages/game-create/index' });
  },
  enterGame() {
    if (!this.data.currentGame) return;
    wx.navigateTo({ url: `/pages/game-detail/index?id=${this.data.currentGame.id}&inviteCode=${this.data.currentGame.inviteCode}` });
  },
  goToRanking() {
    wx.navigateTo({ url: '/pages/ranking/index' });
  },
  showExitTip() {
    wx.showToast({ title: '分值清零后可退出', icon: 'none' });
  },
  async exitGame() {
    if (!this.data.currentGame) return;
    const self = this;
    wx.showModal({
      title: '退出牌局',
      content: '确定要退出当前牌局吗？',
      success: async (res) => {
        if (res.confirm) {
          try {
            await requireLogin();
            await leaveGame(self.data.currentGame!.id);
            wx.showToast({ title: '已退出牌局', icon: 'success' });
            self.onShow(); // Reload
          } catch (err) {
            wx.showToast({ title: (err as any).message || '退出失败', icon: 'none' });
          }
        }
      },
    });
  },
  history() {
    wx.navigateTo({ url: '/pages/history/index' });
  },
});
