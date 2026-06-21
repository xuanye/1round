import { requireLogin } from '../../services/auth.service';
import { reverseScoreTransfer } from '../../services/score.service';
import {
  getSummary,
  getScoreTransfers,
  getSettlementDetail,
  finishGameDirect,
  requestFinish,
  approveFinishRequest,
  rejectFinishRequest,
  getPublicSettlement,
  getJoinMiniProgramCode,
} from '../../services/game.service';
import { formatScore, formatTimeOnly } from '../../utils/format';
import { getUser, saveRecentSession } from '../../utils/storage';
import { RealtimeService } from '../../services/realtime.service';

type ParsedTransferPart = {
  text: string;
  type: 'name' | 'normal' | 'value';
};

function parseTransferText(text: string): ParsedTransferPart[] {
  const parts: ParsedTransferPart[] = [];
  let prefix = '';
  let cleanText = text;
  if (text.startsWith('撤销：')) {
    prefix = '撤销：';
    cleanText = text.substring(3);
  }
  const geiParts = cleanText.split(' 给 ');
  if (geiParts.length < 2) {
    return [{ text, type: 'normal' }];
  }
  const sender = geiParts[0];
  const rest = geiParts[1];
  
  if (prefix) {
    parts.push({ text: prefix, type: 'normal' });
  }
  parts.push({ text: sender, type: 'name' });
  parts.push({ text: ' 给 ', type: 'normal' });
  
  if (rest.indexOf(' 各 +') !== -1) {
    const subparts = rest.split(' 各 +');
    parts.push({ text: subparts[0], type: 'name' });
    parts.push({ text: ' 各 +' + subparts[1], type: 'value' });
  } else if (rest.indexOf(' +') !== -1) {
    const subparts = rest.split(' +');
    parts.push({ text: subparts[0], type: 'name' });
    parts.push({ text: ' +' + subparts[1], type: 'value' });
  } else {
    parts.push({ text: rest, type: 'normal' });
  }
  return parts;
}

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
  receiverPlayerIds: string[];
  transferKind?: string;
  reversalOfTransferId?: string;
  reversedAt?: string;
  canReverse: boolean;
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
      info: '\uf05a',
      undo: '\uf0e2', // Added undo icon code
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
    roundStatus: null as {
      roundNo: number;
      status: string;
      pendingPlayerIds: string[];
      pendingPlayerNames: string[];
      canStartNextRound: boolean;
    } | null,
    uninvolvedNamesText: '',
    myPlayerId: '', // Added myPlayerId

    // Invite Overlay
    showInviteOverlay: false,
    qrCodeUrl: '',
    isLoadingJoinCode: false,

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

      await requireLogin();
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
      setTimeout(() => {
        wx.redirectTo({ url: '/pages/home/index' });
      }, 1500);
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

      const roundStatus = summary.roundStatus || null;
      const uninvolvedNamesText = roundStatus && roundStatus.pendingPlayerNames
        ? roundStatus.pendingPlayerNames.join('、')
        : '';

      const me = participants.find(p => p.isMe);
      const myPlayerId = me ? me.id : '';

      this.setData({
        game: {
          name: summary.name,
          createdAtText: new Date(summary.updatedAt).toLocaleString('zh-CN'),
          inviteCode: summary.inviteCode || this.data.inviteCode,
          isCreator: isCreator,
          status: 'active',
          publicShareToken: summary.publicShareToken || '',
        },
        participants,
        pendingFinishRequest: summary.pendingFinishRequest || null,
        roundStatus,
        uninvolvedNamesText,
        hasMoreTransfers: true,
        myPlayerId,
      });

      // Load first page of transfers
      await this.loadTransfers(true, myPlayerId);
    } catch (err) {
      console.error('Fetch game data failed:', err);
    }
  },

  async loadTransfers(reload = false, currentMyPlayerId?: string) {
    if (this.data.isLoadingTransfers) return;
    if (!reload && !this.data.hasMoreTransfers) return;

    this.setData({ isLoadingTransfers: true });

    let beforeSeq: number | undefined;
    if (!reload && this.data.transfers.length > 0) {
      beforeSeq = this.data.transfers[this.data.transfers.length - 1].sequenceNo;
    }

    const myPlayerId = currentMyPlayerId || this.data.myPlayerId;

    try {
      const list = await getScoreTransfers(this.data.id, beforeSeq, 20);
      const mapped = list.map(t => {
        const timeText = formatTimeOnly(t.createdAt);
        const canReverse =
          this.data.game.status === 'active' &&
          !t.reversedAt &&
          t.transferKind !== 'reversal' &&
          t.receiverPlayerIds.includes(myPlayerId);

        return {
          id: t.id,
          iconCode: t.receiverPlayerIds.length > 1 ? '\uf0c0' : '\uf362',
          text: t.text,
          parsedParts: parseTransferText(t.text),
          time: timeText,
          sequenceNo: t.sequenceNo,
          receiverPlayerIds: t.receiverPlayerIds,
          transferKind: t.transferKind,
          reversalOfTransferId: t.reversalOfTransferId,
          reversedAt: t.reversedAt,
          canReverse: canReverse,
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

      const myPlayer = mappedParticipants.find(p => p.isMe);
      const myPlayerId = myPlayer ? myPlayer.id : '';

      const mappedTransfers = detail.scoreTransfers.map(t => {
        const timeText = formatTimeOnly(t.createdAt);
        return {
          id: t.id,
          iconCode: '\uf362',
          text: t.text,
          parsedParts: parseTransferText(t.text),
          time: timeText,
          sequenceNo: t.sequenceNo,
          receiverPlayerIds: t.receiverPlayerIds || [],
          transferKind: t.transferKind,
          reversalOfTransferId: t.reversalOfTransferId,
          reversedAt: t.reversedAt,
          canReverse: false,
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
        myPlayerId,
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
        const timeText = formatTimeOnly(t.createdAt);
        return {
          id: t.id,
          iconCode: '\uf362',
          text: t.text,
          parsedParts: parseTransferText(t.text),
          time: timeText,
          sequenceNo: t.sequenceNo,
          receiverPlayerIds: t.receiverPlayerIds || [],
          transferKind: t.transferKind,
          reversalOfTransferId: t.reversalOfTransferId,
          reversedAt: t.reversedAt,
          canReverse: false,
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
        myPlayerId: '',
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

  async showInvite() {
    this.setData({
      showInviteOverlay: true,
      isLoadingJoinCode: true,
      qrCodeUrl: '',
    });
    try {
      await requireLogin();
      const qrCodeUrl = await getJoinMiniProgramCode(this.data.id);
      this.setData({ qrCodeUrl, isLoadingJoinCode: false });
    } catch (err) {
      this.setData({ isLoadingJoinCode: false, showInviteOverlay: false });
      wx.showToast({ title: (err as any).message || '生成分享码失败', icon: 'none' });
    }
  },

  hideInvite() {
    this.setData({ showInviteOverlay: false, isLoadingJoinCode: false });
  },

  none() {},

  inputScore() {
    const me = this.data.participants.find(p => p.isMe);
    const roundStatus = this.data.roundStatus;
    if (me && roundStatus && roundStatus.pendingPlayerNames.length > 0) {
      const isMeInvolved = !roundStatus.pendingPlayerIds.includes(me.id);
      if (isMeInvolved) {
        wx.showModal({
          title: '提示',
          content: `当前轮（第 ${roundStatus.roundNo} 局）还有 ${roundStatus.pendingPlayerNames.join('、')} 尚未计分/被计分。你确定要开始新一轮计分吗？`,
          confirmText: '开始新轮',
          cancelText: '取消',
          success: (res) => {
            if (res.confirm) {
              wx.navigateTo({ url: `/pages/score-input/index?id=${this.data.id}` });
            }
          }
        });
        return;
      }
    }
    wx.navigateTo({ url: `/pages/score-input/index?id=${this.data.id}` });
  },

  async onReverseTransfer(e: any) {
    const transferId = e.currentTarget.dataset.transferId;
    if (!transferId) return;

    const self = this;
    wx.showModal({
      title: '确认撤销计分',
      content: '确定要撤销这笔计分吗？撤销后，各接收者的得分将被扣回，并返还发送者的得分。',
      confirmText: '确认撤销',
      confirmColor: '#ba1a1a',
      cancelText: '取消',
      success: async (res) => {
        if (res.confirm) {
          wx.showLoading({ title: '撤销中...' });
          try {
            const idempotencyKey = `reverse_${transferId}`;
            await reverseScoreTransfer(self.data.id, transferId, idempotencyKey, 'user_reversal');
            wx.showToast({ title: '已撤销', icon: 'success' });
            await self.loadGameData();
          } catch (err) {
            console.error('Reverse transfer failed:', err);
            wx.showToast({ title: (err as any).message || '撤销失败', icon: 'none' });
          } finally {
            wx.hideLoading();
          }
        }
      }
    });
  },

  ranking() {
    wx.navigateTo({ url: '/pages/ranking/index' });
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
              await requireLogin();
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
              await requireLogin();
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
      await requireLogin();
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
      await requireLogin();
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
