import { requireLogin } from '../../services/auth.service';
import { getPerformance } from '../../services/game.service';
import type { Performance } from '../../models/game-session';
import { formatScore } from '../../utils/format';
import { chartGeometry, performanceRange, periods, scoreTone } from '../../utils/performance';
import { getSystemFontClass } from '../../utils/system-font';

type ViewPlayer = Performance['players'][number] & { initial: string; scoreLabel: string; scoreTone: string };
type ViewGame = Performance['recentGames'][number] & { meta: string; scoreLabel: string; scoreTone: string };

Page({
  data: {
    fontClass: 'font-system',
    periodIndex: 0,
    periodLabels: periods.map(period => period.label),
    rangeLabel: '',
    start: '', end: '',
    loading: true,
    error: '',
    totalScore: 0, totalScoreLabel: '0', totalScoreTone: 'zero',
    totalGames: 0, wins: 0, maxScore: 0, maxScoreLabel: '0', maxScoreTone: 'zero',
    players: [] as ViewPlayer[], visiblePlayers: [] as ViewPlayer[],
    expanded: false,
    historyItems: [] as ViewGame[],
    trend: [] as Performance['trend'],
    chartWidth: 350,
    chartHeight: 150,
  },

  onShow() {
    this.getTabBar?.()?.setData({ selected: 1 });
    this.setData({ fontClass: getSystemFontClass() });
    void this.loadPerformance();
  },

  async loadPerformance() {
    if (this._loading) return;
    this._loading = true;
    const range = performanceRange(periods[this.data.periodIndex].months);
    this.setData({ loading: true, error: '', rangeLabel: range.label, start: range.start, end: range.end });
    try {
      await requireLogin();
      const performance = await getPerformance(range.start, range.end);
      const players = performance.players.map(player => ({
        ...player, initial: player.displayName.slice(0, 1),
        scoreLabel: formatScore(player.totalScore), scoreTone: scoreTone(player.totalScore),
      }));
      this.setData({
        totalScore: performance.totalScore, totalScoreLabel: formatScore(performance.totalScore), totalScoreTone: scoreTone(performance.totalScore),
        totalGames: performance.totalGames, wins: performance.wins,
        maxScore: performance.maxScore, maxScoreLabel: formatScore(performance.maxScore), maxScoreTone: scoreTone(performance.maxScore),
        trend: performance.trend, players,
        visiblePlayers: this.data.expanded ? players : players.slice(0, 3),
        historyItems: performance.recentGames.map(game => {
          const date = new Date(game.settledAt);
          return { ...game, meta: `${date.getMonth() + 1}月${date.getDate()}日 · ${game.participantCount}人`,
            scoreLabel: formatScore(game.myFinalScore), scoreTone: scoreTone(game.myFinalScore) };
        }),
        loading: false,
      });
      wx.nextTick(() => this.drawChart());
    } catch (err) {
      this.setData({ loading: false, error: '战绩加载失败，请检查网络后重试' });
    } finally {
      this._loading = false;
    }
  },
  _loading: false,

  changePeriod(event: WechatMiniprogram.PickerChange) {
    if (this._loading) return;
    const index = Number(event.detail.value);
    if (!periods[index] || index === this.data.periodIndex) return;
    this.setData({ periodIndex: index, expanded: false });
    void this.loadPerformance();
  },

  togglePlayers() {
    const expanded = !this.data.expanded;
    this.setData({ expanded, visiblePlayers: expanded ? this.data.players : this.data.players.slice(0, 3) });
  },

  drawChart() {
    if (!this.data.totalGames) return;
    wx.createSelectorQuery().in(this).select('.trend-canvas').fields({ node: true, size: true }, rect => {
      if (!rect?.node || !rect.width) return;
      const width = rect.width, height = this.data.chartHeight;
      this.setData({ chartWidth: width });
      const chart = chartGeometry(this.data.trend, this.data.start, this.data.end, width, height);
      const canvas = rect.node as WechatMiniprogram.Canvas;
      const dpr = wx.getWindowInfo().pixelRatio;
      canvas.width = width * dpr;
      canvas.height = height * dpr;
      const ctx = canvas.getContext('2d');
      ctx.scale(dpr, dpr);
      ctx.font = '11px sans-serif';
      ctx.lineWidth = 0.5;
      ctx.strokeStyle = '#DDE3DE';
      ctx.fillStyle = '#68716D';
      ctx.textAlign = 'right';
      ctx.textBaseline = 'middle';
      [chart.top, 0, chart.bottom].forEach(score => {
        const y = chart.y(score);
        ctx.fillText(formatScore(score), chart.left - 8, y);
        ctx.setLineDash([3, 3]);
        ctx.beginPath(); ctx.moveTo(chart.left, y); ctx.lineTo(chart.right, y); ctx.stroke();
      });
      ctx.setLineDash([]);
      ctx.beginPath(); ctx.moveTo(chart.left, chart.yTop); ctx.lineTo(chart.left, chart.yBottom); ctx.stroke();
      ctx.textAlign = 'center';
      chart.labels.forEach(label => ctx.fillText(label.label, Math.min(chart.right - 9, label.x + 8), height - 12));
      ctx.strokeStyle = '#175D50'; ctx.lineWidth = 1.8;
      ctx.beginPath();
      chart.points.forEach((point, index) => index ? ctx.lineTo(point.x, point.y) : ctx.moveTo(point.x, point.y));
      ctx.stroke();
      const last = chart.points[chart.points.length - 1];
      ctx.fillStyle = '#175D50'; ctx.beginPath(); ctx.arc(last.x, last.y, 3, 0, Math.PI * 2); ctx.fill();
      ctx.fillStyle = this.data.totalScoreTone === 'positive' ? '#B95740' : this.data.totalScoreTone === 'negative' ? '#526D65' : '#68716D';
      ctx.textAlign = 'right'; ctx.font = '13px sans-serif';
      ctx.fillText(chart.scoreLabel, Math.max(chart.left + 32, last.x + 8), Math.max(10, last.y - 14));
    }).exec();
  },

  onResize() { wx.nextTick(() => this.drawChart()); },
  async onPullDownRefresh() {
    try { await this.loadPerformance(); } finally { wx.stopPullDownRefresh(); }
  },
  openHistory() { wx.navigateTo({ url: '/pages/history/index' }); },
  openDetail(event: WechatMiniprogram.TouchEvent) {
    const id = String(event.currentTarget.dataset.id || '');
    if (id) wx.navigateTo({ url: `/pages/game-detail/index?id=${encodeURIComponent(id)}` });
  },
});
