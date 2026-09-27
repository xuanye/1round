import { requireLogin } from '../../services/auth.service';
import { getSummary } from '../../services/game.service';
import type { Player } from '../../models/player';
import { RealtimeService } from '../../services/realtime.service';
import { formatScore } from '../../utils/format';
import { scoreTone } from '../../utils/performance';
import { getSystemFontClass } from '../../utils/system-font';

type RankedPlayer = Player & { initial: string; scoreLabel: string; scoreTone: string; isMe: boolean };

Page({
  data: {
    id: '', name: '', fontClass: 'font-system',
    players: [] as RankedPlayer[], loading: true, error: '', finished: false,
  },
  realtime: null as RealtimeService | null,
  _loading: false,
  _visible: false,

  onLoad(query: Record<string, string>) {
    this.setData({ id: query.id || '', fontClass: getSystemFontClass() });
  },
  async onShow() {
    this._visible = true;
    await this.loadRanking();
  },
  onHide() {
    this._visible = false;
    this.realtime?.disconnect();
    this.realtime = null;
  },
  onUnload() { this.onHide(); },

  async loadRanking() {
    if (this._loading) return;
    if (!this.data.id) {
      this.setData({ loading: false, error: '牌局链接无效，请返回牌局重新打开' });
      return;
    }
    this._loading = true;
    this.setData({ loading: true, error: '' });
    try {
      const user = await requireLogin();
      const summary = await getSummary(this.data.id);
      const finished = summary.status === 'finished';
      const players = finished ? [] : [...summary.players]
        .sort((a, b) => b.totalScore - a.totalScore)
        .map(player => ({ ...player, initial: Array.from(player.displayName)[0] || '',
          scoreLabel: formatScore(player.totalScore), scoreTone: scoreTone(player.totalScore),
          isMe: player.userId === user.id }));
      this.setData({ name: summary.name, players, finished, loading: false });
      if (finished) this.realtime?.disconnect();
      else if (this._visible && !this.realtime) {
        this.realtime = new RealtimeService();
        this.realtime.onEvent(() => { void this.loadRanking(); });
        this.realtime.connect(this.data.id);
      }
    } catch {
      this.setData({ loading: false, error: '本局排名加载失败，请检查网络或返回牌局后重试' });
    } finally {
      this._loading = false;
    }
  },
  async onPullDownRefresh() {
    try { await this.loadRanking(); } finally { wx.stopPullDownRefresh(); }
  },
  returnToGame() { wx.switchTab({ url: '/pages/home/index' }); },
});
