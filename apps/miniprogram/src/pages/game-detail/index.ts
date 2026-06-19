import { ensureLogin } from '../../services/auth.service';
import {
  getSummary,
  getScoreTransfers,
  getSettlementDetail,
  finishGameDirect,
  requestFinish,
  approveFinishRequest,
  rejectFinishRequest,
  getPublicSettlement,
} from '../../services/game.service';
import { getUser, saveRecentSession } from '../../utils/storage';
import { RealtimeService } from '../../services/realtime.service';
import { drawQRCode } from '../../utils/qrcode';
import { formatScore } from '../../utils/format';

type DetailParticipant = {
  id: string;
  initial: string;
  name: string;
  role: string;
  score: string;
  scoreTone: 'positive' | 'negative' | 'muted';
  isCreator: boolean;
  isMe: boolean;
};

type DetailTransfer = {
  id: string;
  iconCode: string;
  text: string;
  time: string;
  sequenceNo: number;
};

Page({
  data: {
    icons: {
      back: '\uf060',
      qrCode: '\uf029',
      ranking: '\ue561',
      star: '\uf005',
      plusCircle: '\uf055',
      history: '\uf1da',
      flag: '\uf024',
      home: '\uf015',
      chart: '\uf201',
    },
    id: '',
    inviteCode: '',
    shareToken: '',

    game: {
      name: '加载中...',
      createdAtText: '',
      inviteCode: '',
      isCreator: false,
      status: 'active' as 'active' | 'finished',
      publicShareToken: '',
    },
    participants: [] as DetailParticipant[],
    transfers: [] as DetailTransfer[],
    pendingFinishRequest: null as {
      id: string;
      requestedByPlayerId: string;
      requestedByName: string;
      createdAt: string;
    } | null,

    // Invite Overlay
    showInviteOverlay: false,
    qrCodeUrl: '',

    // Pagination
    hasMoreTransfers: true,
    isLoadingTransfers: false,
  },

  realtime: null as RealtimeService | null,

  async onLoad(query: Record<string, string>) {
    const id = query.id || '';
    const inviteCode = query.inviteCode || '';
    const shareToken = query.shareToken || '';
    this.setData({ id, inviteCode, shareToken });

    if (id) {
      this.realtime = new RealtimeService();
    }
  },

  async onShow() {
    wx.showLoading({ title: '加载中...' });
    try {
      // If opened via shareToken, we don't force login since public settlements are unauthenticated
      if (this.data.shareToken) {
        await this.loadPublicSettlement();
        return;
      }

      await ensureLogin();
      await this.loadGameData();

      // Connect to websocket if the game is active
      if (this.data.game.status === 'active' && this.realtime) {
        this.realtime.connect(this.data.id);
        this.realtime.onEvent(() => {
          // Re-fetch everything on socket notifications
          this.loadGameData();
        });
      }
    } catch (err) {
      console.error('Game detail load failed:', err);
      wx.showToast({ title: (err as any).message || '加载失败', icon: 'none' });
    } finally {
      wx.hideLoading();
    }
  },

  onHide() {
    if (this.realtime) {
      this.realtime.disconnect();
    }
  },

  onUnload() {
    if (this.realtime) {
      this.realtime.disconnect();
    }
  },

  async loadGameData() {
    const id = this.data.id;
    try {
      const user = getUser();
      const summary = await getSummary(id);
      saveRecentSession(id);

      // Determine creator using robust ownerUserId field
      const isCreator = summary.ownerUserId === user?.id;

      const isFinished = summary.status === 'finished';

      if (isFinished) {
        if (this.realtime) this.realtime.disconnect();
        await this.loadSettledGame();
        return;
      }

      const participants = summary.players.map((p) => {
        const isMe = p.userId === user?.id || (p.displayName === user?.displayName && p.userId === user?.id);
        const isPlayerCreator = p.userId === summary.ownerUserId;
        return {
          id: p.id,
          initial: p.displayName.slice(0, 1),
          name: p.displayName,
          role: isPlayerCreator ? '创建者' : '已加入',
          score: formatScore(p.totalScore),
          scoreTone: p.totalScore > 0 ? 'positive' as const : p.totalScore < 0 ? 'negative' as const : 'muted' as const,
          isCreator: isPlayerCreator,
          isMe: isMe,
        };
      });

      this.setData({
        game: {
          name: summary.name,
          createdAtText: new Date(summary.updatedAt).toLocaleString('zh-CN'),
          inviteCode: this.data.inviteCode,
          isCreator: isCreator,
          status: 'active',
          publicShareToken: summary.publicShareToken || '',
        },
        participants,
        pendingFinishRequest: summary.pendingFinishRequest || null,
        hasMoreTransfers: true,
      });

      // Load first page of transfers
      await this.loadTransfers(true);
    } catch (err) {
      console.error('Fetch game data failed:', err);
    }
  },

  async loadTransfers(reload = false) {
    if (this.data.isLoadingTransfers) return;
    if (!reload && !this.data.hasMoreTransfers) return;

    this.setData({ isLoadingTransfers: true });

    let beforeSeq: number | undefined;
    if (!reload && this.data.transfers.length > 0) {
      beforeSeq = this.data.transfers[this.data.transfers.length - 1].sequenceNo;
    }

    try {
      const list = await getScoreTransfers(this.data.id, beforeSeq, 20);
      const mapped = list.map(t => {
        const timeText = new Date(t.createdAt).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' });
        return {
          id: t.id,
          iconCode: t.receiverPlayerIds.length > 1 ? '\uf0c0' : '\uf362',
          text: t.text,
          time: timeText,
          sequenceNo: t.sequenceNo,
        };
      });

      const nextTransfers = reload ? mapped : [...this.data.transfers, ...mapped];
      this.setData({
        transfers: nextTransfers,
        hasMoreTransfers: list.length === 20,
        isLoadingTransfers: false,
      });
    } catch (err) {
      console.error('Fetch transfers failed:', err);
      this.setData({ isLoadingTransfers: false });
    }
  },

  async loadSettledGame() {
    try {
      const detail = await getSettlementDetail(this.data.id);
      saveRecentSession(this.data.id);
      const user = getUser();
      const mappedParticipants = detail.participants.map(p => ({
        id: p.id,
        initial: p.displayName.slice(0, 1),
        name: p.displayName,
        role: '已结算',
        score: formatScore(p.finalScore),
        scoreTone: p.finalScore > 0 ? 'positive' as const : p.finalScore < 0 ? 'negative' as const : 'muted' as const,
        isCreator: false,
        isMe: p.displayName === user?.displayName,
      }));

      const mappedTransfers = detail.scoreTransfers.map(t => {
        const timeText = new Date(t.createdAt).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' });
        return {
          id: t.id,
          iconCode: '\uf362',
          text: t.text,
          time: timeText,
          sequenceNo: t.sequenceNo,
        };
      });

      this.setData({
        game: {
          name: detail.name,
          createdAtText: new Date(detail.settledAt).toLocaleString('zh-CN'),
          inviteCode: '',
          isCreator: false,
          status: 'finished',
          publicShareToken: detail.publicShareToken || '',
        },
        participants: mappedParticipants,
        transfers: mappedTransfers,
        pendingFinishRequest: null,
        hasMoreTransfers: false,
      });
    } catch (err) {
      console.error('Load settled game failed:', err);
    }
  },

  async loadPublicSettlement() {
    try {
      const detail = await getPublicSettlement(this.data.shareToken);
      const mappedParticipants = detail.participants.map(p => ({
        id: p.id,
        initial: p.displayName.slice(0, 1),
        name: p.displayName,
        role: '公开结算',
        score: `${p.finalScore >= 0 ? '+' : ''}${p.finalScore}`,
        scoreTone: p.finalScore > 0 ? 'positive' as const : p.finalScore < 0 ? 'negative' as const : 'muted' as const,
        isCreator: false,
        isMe: false,
      }));

      const mappedTransfers = detail.scoreTransfers.map(t => {
        const timeText = new Date(t.createdAt).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' });
        return {
          id: t.id,
          iconCode: '\uf362',
          text: t.text,
          time: timeText,
          sequenceNo: t.sequenceNo,
        };
      });

      this.setData({
        game: {
          name: detail.name,
          createdAtText: new Date(detail.settledAt).toLocaleDateString('zh-CN'),
          inviteCode: '',
          isCreator: false,
          status: 'finished',
          publicShareToken: this.data.shareToken,
        },
        participants: mappedParticipants,
        transfers: mappedTransfers,
        pendingFinishRequest: null,
        hasMoreTransfers: false,
      });
    } catch (err) {
      console.error('Load public settlement failed:', err);
    }
  },

  async onReachBottom() {
    if (this.data.game.status === 'active') {
      await this.loadTransfers();
    }
  },

  backHome() {
    wx.redirectTo({ url: '/pages/home/index' });
  },

  showInvite() {
    const inviteLink = `https://oneround.app/join?code=${this.data.inviteCode}`;
    this.setData({
      showInviteOverlay: true,
    }, () => {
      wx.nextTick(() => {
        drawQRCode('inviteQR', inviteLink, 180, this);
      });
    });
  },

  hideInvite() {
    this.setData({ showInviteOverlay: false });
  },

  none() {},

  inputScore() {
    wx.navigateTo({ url: `/pages/score-input/index?id=${this.data.id}` });
  },

  ranking() {
    wx.navigateTo({ url: `/pages/ranking/index?id=${this.data.id}` });
  },

  renameSelf() {
    const me = this.data.participants.find(p => p.isMe);
    if (!me) return;
    wx.navigateTo({ url: `/pages/player-manage/index?id=${this.data.id}&displayName=${encodeURIComponent(me.name)}` });
  },

  finish() {
    const self = this;
    if (this.data.game.isCreator) {
      wx.showModal({
        title: '结束牌局',
        content: '确定要直接结束本局并进行冻结结算吗？',
        confirmText: '确认结束',
        confirmColor: '#ba1a1a',
        success: async (res) => {
          if (res.confirm) {
            try {
              await finishGameDirect(self.data.id);
              wx.showToast({ title: '牌局已结束', icon: 'success' });
              self.loadGameData();
            } catch (err) {
              wx.showToast({ title: (err as any).message || '操作失败', icon: 'none' });
            }
          }
        },
      });
    } else {
      wx.showModal({
        title: '申请结束牌局',
        content: '确定要向创建者发起结束牌局的申请吗？',
        success: async (res) => {
          if (res.confirm) {
            try {
              await requestFinish(self.data.id);
              wx.showToast({ title: '已发起申请', icon: 'success' });
              self.loadGameData();
            } catch (err) {
              wx.showToast({ title: (err as any).message || '发起失败', icon: 'none' });
            }
          }
        },
      });
    }
  },

  async approveFinish() {
    if (!this.data.pendingFinishRequest) return;
    try {
      await approveFinishRequest(this.data.id, this.data.pendingFinishRequest.id);
      wx.showToast({ title: '已同意结束', icon: 'success' });
      this.loadGameData();
    } catch (err) {
      wx.showToast({ title: (err as any).message || '操作失败', icon: 'none' });
    }
  },

  async rejectFinish() {
    if (!this.data.pendingFinishRequest) return;
    try {
      await rejectFinishRequest(this.data.id, this.data.pendingFinishRequest.id);
      wx.showToast({ title: '已拒绝结束', icon: 'success' });
      this.loadGameData();
    } catch (err) {
      wx.showToast({ title: (err as any).message || '操作失败', icon: 'none' });
    }
  },

  // Share card config for settled game
  onShareAppMessage() {
    const isFinished = this.data.game.status === 'finished';
    if (isFinished) {
      return {
        title: `【一局一分】牌局结算：“${this.data.game.name}”`,
        path: `/pages/game-detail/index?shareToken=${this.data.game.publicShareToken}`, // sharing public settlement page
      };
    }
    return {
      title: '一局一分：邀请你加入牌局',
      path: `/pages/game-join/index?inviteCode=${this.data.inviteCode}`,
    };
  },
});
