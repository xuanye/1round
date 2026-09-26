import { requireLogin } from '../../services/auth.service';
import { getCurrentGame, getHistory, leaveGame } from '../../services/game.service';
import { formatFriendlyTime, formatScore } from '../../utils/format';
import { RealtimeService } from '../../services/realtime.service';
import { createGameDetailPage } from '../game-detail/page';
import { getSystemFontClass } from '../../utils/system-font';

function extractInviteCode(scanResult: WechatMiniprogram.ScanCodeSuccessCallbackResult): string {
  const candidates = [scanResult.path, scanResult.result].filter(Boolean) as string[];
  for (const candidate of candidates) {
    const inviteCodeMatch = candidate.match(/inviteCode=([A-Za-z0-9]+)/);
    if (inviteCodeMatch) return inviteCodeMatch[1];

    const sceneMatch = candidate.match(/(?:^|[?&])scene=([^&#]+)/);
    if (sceneMatch) {
      const scene = decodeURIComponent(sceneMatch[1]);
      const codeMatch = scene.match(/^(?:code=)?([A-Za-z0-9]+)$/);
      if (codeMatch) return codeMatch[1];
    }

    const codeMatch = candidate.match(/(?:^|[?&])code=([A-Za-z0-9]+)/);
    if (codeMatch) return codeMatch[1];

    const rawCodeMatch = candidate.match(/^[A-Za-z0-9]{4,}$/);
    if (rawCodeMatch) return rawCodeMatch[0];
  }
  return '';
}

const detail = createGameDetailPage();

const home: WechatMiniprogram.Page.Options<any, any> = {
  loading: false,
  visible: false,
  data: Object.assign({}, detail.data, {
    isHome: true,
    fontClass: 'font-system',
    userName: '',
    homeState: 'loading' as 'loading' | 'ready' | 'error',
    homeError: '',
    recentError: '',
    recentGames: [] as { id: string; title: string; meta: string; score: string; scoreTone: string }[],
  }),

  onLoad() {},

  async onShow() {
    this.setData({ fontClass: getSystemFontClass() });
    this.getTabBar?.()?.setData({ selected: 0 });
    this.visible = true;
    await this.refreshHome();
  },

  async refreshHome() {
    if (this.loading) return;
    this.loading = true;
    this.realtime?.disconnect();
    this.setData({ homeState: 'loading', homeError: '', showInviteOverlay: false });
    wx.showLoading({ title: '加载中...' });
    try {
      const user = await requireLogin();
      const current = await getCurrentGame();
      this.setData({ userName: user.displayName || '老书记' });
      if (current?.id) {
        if (current.id !== this.data.id) {
          this.setData({ participants: [], transfers: [], pendingFinishRequest: null, roundStatus: null });
        }
        this.setData({ id: current.id, inviteCode: current.inviteCode, shareToken: '', isPublicShare: false });
        const loaded = await this.loadGameData();
        if (!loaded) throw new Error('牌局加载失败，请重试');
        this.setData({ homeState: 'ready' });
        if (this.visible && this.data.game.status === 'active') {
          this.realtime = new RealtimeService();
          this.realtime.onEvent(() => { this.loadGameData(); });
          this.realtime.connect(current.id);
        }
      } else {
        this.setData({ id: '', participants: [], transfers: [], homeState: 'ready', recentGames: [] });
        await this.loadRecentGames();
      }
    } catch (err) {
      this.setData({ homeState: 'error', homeError: (err as Error).message || '加载失败，请重试' });
    } finally {
      this.loading = false;
      wx.hideLoading();
    }
  },

  onHide() {
    this.visible = false;
    this.realtime?.disconnect();
  },

  async loadRecentGames() {
    this.setData({ recentError: '' });
    try {
      const history = await getHistory('', 5);
      this.setData({ recentGames: history.items.map(item => ({
        id: item.id,
        title: item.name,
        meta: `${formatFriendlyTime(item.settledAt)} · ${item.participantCount || 0}人 · ${item.scoreTransferCount}笔计分`,
        score: formatScore(item.myFinalScore),
        scoreTone: item.myFinalScore > 0 ? 'positive' : item.myFinalScore < 0 ? 'negative' : 'zero',
      })) });
    } catch (err) {
      this.setData({ recentError: '最近牌局加载失败，点击重试' });
    }
  },

  async onPullDownRefresh() {
    try { await this.refreshHome(); } finally { wx.stopPullDownRefresh(); }
  },

  async onReachBottom() {
    if (this.data.homeState === 'ready' && this.data.id) await detail.onReachBottom!.call(this);
  },

  createGame() {
    wx.navigateTo({ url: '/pages/game-create/index' });
  },
  scanToJoin() {
    wx.scanCode({
      onlyFromCamera: false,
      success: (res) => {
        const inviteCode = extractInviteCode(res);
        if (!inviteCode) {
          wx.showToast({ title: '未识别到邀请码', icon: 'none' });
          return;
        }
        wx.navigateTo({ url: `/pages/game-join/index?inviteCode=${inviteCode}` });
      },
      fail: (err) => {
        if (err.errMsg?.includes('cancel')) return;
        wx.showToast({ title: '扫码失败，请重试', icon: 'none' });
      },
    });
  },

  history() {
    wx.navigateTo({ url: '/pages/history/index' });
  },

  openRecent(event: WechatMiniprogram.TouchEvent) {
    const id = String(event.currentTarget.dataset.id);
    wx.navigateTo({ url: `/pages/game-detail/index?id=${id}` });
  },

  exitGame() {
    const me = this.data.participants.find(p => p.isMe);
    if (!me || me.score !== '0') {
      wx.showToast({ title: '当前分值不为 0，暂时不能退出', icon: 'none' });
      return;
    }
    wx.showModal({
      title: '退出牌局',
      content: '确定要退出当前牌局吗？',
      success: async (res) => {
        if (!res.confirm) return;
        try {
          await requireLogin();
          await leaveGame(this.data.id);
          wx.showToast({ title: '已退出牌局', icon: 'success' });
          await this.refreshHome();
        } catch (err) {
          wx.showToast({ title: (err as Error).message || '退出失败', icon: 'none' });
        }
      },
    });
  },
};

Page(Object.assign(detail, home));
