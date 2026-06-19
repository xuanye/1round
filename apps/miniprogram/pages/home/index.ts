Page({
  data: {
    userName: '老书记01',
    currentGame: {
      id: 'mock-game-001',
      name: '周末家庭聚会',
      date: '2024-05-20',
      participantCount: 4,
      myScore: 36,
      canExit: false,
    },
    stats: [
      { label: '聚会次数', value: '12', unit: '场', tone: 'primary', icon: '◎' },
      { label: '最高得分', value: '+128', unit: '', tone: 'tertiary', icon: '★' },
    ],
    recentGames: [
      {
        title: '五一快乐麻将',
        meta: '昨天 · 5人 · 18次计分',
        status: '已结束',
        score: '+12',
        icon: '麻',
      },
      {
        title: '春节斗地主',
        meta: '2024-02-10 · 3人 · 24次计分',
        status: '已结束',
        score: '+84',
        icon: '斗',
      },
    ],
  },
  createGame() {
    wx.navigateTo({ url: '/pages/game-create/index' });
  },
  enterGame() {
    wx.navigateTo({ url: `/pages/game-detail/index?id=${this.data.currentGame.id}` });
  },
  joinGame() {
    wx.navigateTo({ url: '/pages/game-join/index' });
  },
  showExitTip() {
    wx.showToast({ title: '分值清零后可退出', icon: 'none' });
  },
  history() {
    wx.navigateTo({ url: '/pages/history/index' });
  },
});
