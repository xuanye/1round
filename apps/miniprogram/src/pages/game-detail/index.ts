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
  getSettlementMiniProgramCode,
} from '../../services/game.service';
import { formatScore, formatTimeOnly } from '../../utils/format';
import { getUser, saveRecentSession } from '../../utils/storage';
import { RealtimeService } from '../../services/realtime.service';
import type { ScoreTransfer, ScoreChange } from '../../models/score-transfer';

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
  text: string;
  parsedParts: ParsedTransferPart[];
  time: string;
  sequenceNo: number;
  receiverPlayerIds: string[];
  transferKind?: string;
  reversalOfTransferId?: string;
  reversedAt?: string;
  canReverse: boolean;
  isReversal: boolean;
  originalText: string;
  initiatedByName: string;
  initiatorInitial: string;
  senderChange: DetailScoreChange | null;
  scoreChanges: DetailScoreChange[];
  changeLabel: string;
};

type DetailScoreChange = {
  playerId: string;
  playerName: string;
  initial: string;
  beforeText: string;
  afterText: string;
  deltaText: string;
  effectText: string;
  effectTone: 'returned' | 'deducted';
  afterTone: 'positive' | 'negative' | 'muted';
};

function formatTransferTime(createdAt: string): string {
  const date = new Date(createdAt);
  if (isNaN(date.getTime())) return '';
  const now = new Date();
  const time = formatTimeOnly(date);
  if (date.toDateString() === now.toDateString()) return `今天 ${time}`;
  const yesterday = new Date(now.getFullYear(), now.getMonth(), now.getDate() - 1);
  if (date.toDateString() === yesterday.toDateString()) return `昨天 ${time}`;
  const dateText = date.getFullYear() === now.getFullYear()
    ? `${date.getMonth() + 1}月${date.getDate()}日`
    : `${date.getFullYear()}年${date.getMonth() + 1}月${date.getDate()}日`;
  return `${dateText} ${time}`;
}

function mapScoreChange(change: ScoreChange): DetailScoreChange {
  return {
    playerId: change.playerId,
    playerName: change.playerName,
    initial: change.playerName.slice(0, 1),
    beforeText: formatScore(change.before),
    afterText: formatScore(change.after),
    deltaText: formatScore(change.delta),
    effectText: change.delta > 0 ? `返还 ${formatScore(change.delta)}` : `扣回 ${Math.abs(change.delta)}`,
    effectTone: change.delta > 0 ? 'returned' : 'deducted',
    afterTone: change.after > 0 ? 'positive' : change.after < 0 ? 'negative' : 'muted',
  };
}

function mapDetailTransfer(transfer: ScoreTransfer, canReverse: boolean): DetailTransfer {
  const isReversal = transfer.transferKind === 'reversal';
  const originalText = transfer.text.replace(/^撤销：/, '').replace(/ \(已撤销\)$/, '');
  const scoreChanges = (transfer.scoreChanges || []).map(mapScoreChange);
  const initiatedByName = transfer.initiatedByName || scoreChanges[0]?.playerName || '';
  return {
    id: transfer.id,
    text: transfer.text,
    parsedParts: parseTransferText(originalText),
    originalText,
    time: formatTransferTime(transfer.createdAt),
    sequenceNo: transfer.sequenceNo,
    receiverPlayerIds: transfer.receiverPlayerIds || [],
    transferKind: transfer.transferKind,
    reversalOfTransferId: transfer.reversalOfTransferId,
    reversedAt: transfer.reversedAt,
    canReverse,
    isReversal,
    initiatedByName,
    initiatorInitial: initiatedByName.slice(0, 1),
    senderChange: scoreChanges[0] || null,
    scoreChanges,
    changeLabel: isReversal ? '撤销后的积分变化' : transfer.reversedAt ? '原计分时的发起方积分' : '发起方积分',
  };
}

function parseShareToken(query: Record<string, string>): string {
  if (query.shareToken) return query.shareToken;
  if (!query.scene) return '';

  const scene = decodeURIComponent(query.scene);
  const match = scene.match(/(?:^|&)shareToken=([^&]+)/);
  return match ? match[1] : '';
}

function truncatePosterText(text: string, maxChars: number): string {
  return text.length > maxChars ? `${text.slice(0, maxChars - 1)}...` : text;
}

function formatDateOnly(dateInput: Date | string | number): string {
  const d = new Date(dateInput);
  if (isNaN(d.getTime())) return '';
  const year = d.getFullYear();
  const month = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  return `${year}/${month}/${day}`;
}

function scoreColor(tone: DetailParticipant['scoreTone']): string {
  if (tone === 'positive') return '#1F695D';
  if (tone === 'negative') return '#BA1A1A';
  return '#3F4946';
}

function drawRoundRect(ctx: WechatMiniprogram.CanvasContext, x: number, y: number, width: number, height: number, radius: number): void {
  ctx.beginPath();
  ctx.moveTo(x + radius, y);
  ctx.lineTo(x + width - radius, y);
  ctx.arcTo(x + width, y, x + width, y + radius, radius);
  ctx.lineTo(x + width, y + height - radius);
  ctx.arcTo(x + width, y + height, x + width - radius, y + height, radius);
  ctx.lineTo(x + radius, y + height);
  ctx.arcTo(x, y + height, x, y + height - radius, radius);
  ctx.lineTo(x, y + radius);
  ctx.arcTo(x, y, x + radius, y, radius);
  ctx.closePath();
}

function fillRoundRect(ctx: WechatMiniprogram.CanvasContext, x: number, y: number, width: number, height: number, radius: number, color: string): void {
  drawRoundRect(ctx, x, y, width, height, radius);
  ctx.setFillStyle(color);
  ctx.fill();
}

function strokeRoundRect(ctx: WechatMiniprogram.CanvasContext, x: number, y: number, width: number, height: number, radius: number, color: string, lineWidth = 1): void {
  drawRoundRect(ctx, x, y, width, height, radius);
  ctx.setStrokeStyle(color);
  ctx.setLineWidth(lineWidth);
  ctx.stroke();
}

function fillCircle(ctx: WechatMiniprogram.CanvasContext, x: number, y: number, radius: number, color: string): void {
  ctx.beginPath();
  ctx.arc(x, y, radius, 0, Math.PI * 2);
  ctx.closePath();
  ctx.setFillStyle(color);
  ctx.fill();
}

function drawBoldText(ctx: WechatMiniprogram.CanvasContext, text: string, x: number, y: number, color: string, size: number): void {
  ctx.setFillStyle(color);
  ctx.setFontSize(size);
  ctx.fillText(text, x, y);
  ctx.fillText(text, x + 1, y);
}

function drawPosterLogo(ctx: WechatMiniprogram.CanvasContext, x: number, y: number): void {
  fillCircle(ctx, x, y, 44, '#FFFCF6');
  ctx.save();
  ctx.translate(x, y);
  ctx.rotate(-0.18);
  drawBoldText(ctx, 'z', -18, 25, '#00604F', 72);
  ctx.restore();
}

function drawGlobeIcon(ctx: WechatMiniprogram.CanvasContext, x: number, y: number): void {
  ctx.setStrokeStyle('#FFFFFF');
  ctx.setLineWidth(3);
  ctx.beginPath();
  ctx.arc(x, y, 15, 0, Math.PI * 2);
  ctx.moveTo(x - 15, y);
  ctx.lineTo(x + 15, y);
  ctx.moveTo(x, y - 15);
  ctx.lineTo(x, y + 15);
  ctx.moveTo(x - 10, y - 10);
  ctx.quadraticCurveTo(x, y - 2, x + 10, y - 10);
  ctx.moveTo(x - 10, y + 10);
  ctx.quadraticCurveTo(x, y + 2, x + 10, y + 10);
  ctx.stroke();
}

function drawRankBadge(ctx: WechatMiniprogram.CanvasContext, rank: number, x: number, y: number): void {
  const fills = ['#D99A3D', '#B8B8B8', '#B97842'];
  const fill = fills[rank - 1] || '#E9EEE9';
  const text = rank <= 3 ? '#FFFFFF' : '#1F695D';
  fillCircle(ctx, x, y, 24, fill);
  ctx.setFillStyle(text);
  ctx.setFontSize(24);
  ctx.setTextAlign('center');
  ctx.fillText(String(rank), x, y + 8);
  ctx.setTextAlign('left');
}

function drawDashedLine(ctx: WechatMiniprogram.CanvasContext, x1: number, y: number, x2: number, dash = 10, gap = 8): void {
  ctx.beginPath();
  for (let x = x1; x < x2; x += dash + gap) {
    ctx.moveTo(x, y);
    ctx.lineTo(Math.min(x + dash, x2), y);
  }
  ctx.stroke();
}

function drawVerticalDashedLine(ctx: WechatMiniprogram.CanvasContext, x: number, y1: number, y2: number, dash = 8, gap = 8): void {
  ctx.beginPath();
  for (let y = y1; y < y2; y += dash + gap) {
    ctx.moveTo(x, y);
    ctx.lineTo(x, Math.min(y + dash, y2));
  }
  ctx.stroke();
}

function drawMedalBadge(ctx: WechatMiniprogram.CanvasContext, rank: number, x: number, y: number): void {
  const fills = ['#D99A3D', '#B8B8B8', '#B97842'];
  const fill = fills[rank - 1] || '#E9EEE9';
  ctx.setFillStyle(fill);
  ctx.beginPath();
  ctx.moveTo(x - 18, y + 26);
  ctx.lineTo(x - 30, y + 66);
  ctx.lineTo(x, y + 48);
  ctx.lineTo(x + 30, y + 66);
  ctx.lineTo(x + 18, y + 26);
  ctx.closePath();
  ctx.fill();
  fillCircle(ctx, x, y, 38, fill);
  strokeRoundRect(ctx, x - 29, y - 29, 58, 58, 29, 'rgba(255, 255, 255, 0.36)', 3);
  ctx.setFillStyle('#FFFFFF');
  ctx.setFontSize(44);
  ctx.setTextAlign('center');
  ctx.fillText(String(rank), x, y + 16);
  ctx.setTextAlign('left');
}

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
      undo: '\uf0e2',
    },
    id: '',
    inviteCode: '',
    shareToken: '',
    isPublicShare: false,

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
    myPlayerId: '',

    // Invite Overlay
    showInviteOverlay: false,
    qrCodeUrl: '',
    isLoadingJoinCode: false,

    // Pagination
    hasMoreTransfers: true,
    isLoadingTransfers: false,
    isSavingPoster: false,
  },

  realtime: null as RealtimeService | null,

  async onLoad(query: Record<string, string>) {
    const id = query.id || '';
    const inviteCode = query.inviteCode || '';
    const shareToken = parseShareToken(query);
    this.setData({ id, inviteCode, shareToken, isPublicShare: !!shareToken });

    if (id) {
      this.realtime = new RealtimeService();
    }
  },

  async onShow() {
    wx.showLoading({ title: '加载中...' });
    try {
      if (this.data.shareToken) {
        await this.loadPublicSettlement();
        return;
      }
      if (!this.data.id) {
        wx.showToast({ title: '结算链接无效', icon: 'none' });
        setTimeout(() => {
          wx.redirectTo({ url: '/pages/home/index' });
        }, 1500);
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
        const canReverse =
          this.data.game.status === 'active' &&
          !t.reversedAt &&
          t.transferKind !== 'reversal' &&
          t.receiverPlayerIds.includes(myPlayerId);
        return mapDetailTransfer(t, canReverse);
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

      const mappedTransfers = detail.scoreTransfers.map(t => mapDetailTransfer(t, false));

      this.setData({
        game: {
          name: detail.name,
          createdAtText: formatDateOnly(detail.settledAt),
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
        score: formatScore(p.finalScore),
        scoreTone: p.finalScore > 0 ? 'positive' as const : p.finalScore < 0 ? 'negative' as const : 'muted' as const,
        isCreator: false,
        isMe: false,
      }));

      this.setData({
        game: {
          name: detail.name,
          createdAtText: formatDateOnly(detail.settledAt),
          inviteCode: '',
          isCreator: false,
          status: 'finished',
          publicShareToken: this.data.shareToken,
        },
        participants: mappedParticipants,
        transfers: [],
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

  enterMiniProgram() {
    wx.redirectTo({ url: '/pages/home/index' });
  },

  async saveSettlementPoster() {
    if (this.data.isSavingPoster) return;
    if (!this.data.id || this.data.game.status !== 'finished') {
      wx.showToast({ title: '结算信息不可用', icon: 'none' });
      return;
    }

    this.setData({ isSavingPoster: true });
    wx.showLoading({ title: '生成图片...' });
    try {
      const codePath = await getSettlementMiniProgramCode(this.data.id);
      const posterPath = await this.drawSettlementPoster(codePath);
      await this.saveImageToAlbum(posterPath);
      wx.showToast({ title: '已保存到相册', icon: 'success' });
    } catch (err) {
      console.error('Save settlement poster failed:', err);
      wx.showToast({ title: (err as any).message || '保存失败', icon: 'none' });
    } finally {
      wx.hideLoading();
      this.setData({ isSavingPoster: false });
    }
  },

  drawSettlementPoster(codePath: string): Promise<string> {
    return new Promise((resolve, reject) => {
      const width = 926;
      const height = 1800;
      const ctx = wx.createCanvasContext('settlementPoster', this);
      const topParticipant = this.data.participants[0];
      const rankingParticipants = this.data.participants.slice(1, 4);

      ctx.setFillStyle('#F8F4EC');
      ctx.fillRect(0, 0, width, height);

      fillRoundRect(ctx, 58, 70, 810, 1670, 30, 'rgba(72, 48, 18, 0.08)');
      fillRoundRect(ctx, 48, 56, 830, 1688, 34, '#FFFCF6');

      fillRoundRect(ctx, 48, 56, 830, 650, 34, '#00604F');
      fillRoundRect(ctx, 48, 56, 830, 158, 34, '#0B594D');

      ctx.setFillStyle('rgba(255, 255, 255, 0.08)');
      ctx.setFontSize(410);
      ctx.setTextAlign('right');
      ctx.fillText('1', 850, 580);
      ctx.setTextAlign('left');

      ctx.setFillStyle('#FFFCF6');
      ctx.beginPath();
      ctx.moveTo(48, 636);
      ctx.quadraticCurveTo(464, 746, 878, 608);
      ctx.lineTo(878, 706);
      ctx.lineTo(48, 706);
      ctx.closePath();
      ctx.fill();

      ctx.setStrokeStyle('#D99A3D');
      ctx.setLineWidth(5);
      ctx.beginPath();
      ctx.moveTo(48, 636);
      ctx.quadraticCurveTo(464, 746, 878, 608);
      ctx.stroke();

      drawPosterLogo(ctx, 146, 148);
      drawBoldText(ctx, '一局一分', 198, 164, '#FFFFFF', 44);

      fillRoundRect(ctx, 632, 118, 196, 62, 31, '#B8742C');
      drawGlobeIcon(ctx, 665, 149);
      drawBoldText(ctx, '公开结算', 700, 160, '#FFFFFF', 30);

      ctx.setStrokeStyle('rgba(255, 255, 255, 0.36)');
      ctx.setLineWidth(2);
      drawDashedLine(ctx, 104, 232, 824, 12, 8);

      drawBoldText(ctx, truncatePosterText(this.data.game.name, 7), 104, 350, '#FFFFFF', 72);
      ctx.setFillStyle('rgba(255, 255, 255, 0.76)');
      ctx.setFontSize(34);
      ctx.fillText(`${this.data.game.createdAtText} 已结算`, 104, 410);

      if (topParticipant) {
        fillRoundRect(ctx, 104, 448, 512, 106, 16, 'rgba(0, 0, 0, 0.10)');
        strokeRoundRect(ctx, 104, 448, 512, 106, 16, 'rgba(255, 210, 148, 0.54)', 2);
        fillCircle(ctx, 168, 501, 36, '#F4B968');
        ctx.setFillStyle('#00604F');
        ctx.setFontSize(34);
        ctx.fillText('★', 152, 513);
        ctx.setFillStyle('#F4B968');
        ctx.setFontSize(30);
        ctx.fillText('本局最高', 230, 490);
        ctx.setFillStyle('#FFFFFF');
        ctx.setFontSize(44);
        ctx.fillText(truncatePosterText(topParticipant.name, 5), 230, 534);
        ctx.setStrokeStyle('rgba(255, 255, 255, 0.28)');
        ctx.setLineWidth(2);
        ctx.beginPath();
        ctx.moveTo(394, 475);
        ctx.lineTo(394, 530);
        ctx.stroke();
        ctx.setFillStyle('#F4B968');
        ctx.setFontSize(76);
        ctx.fillText(topParticipant.score, 430, 526);
      }

      fillRoundRect(ctx, 92, 714, 740, 560, 24, 'rgba(86, 55, 22, 0.08)');
      fillRoundRect(ctx, 92, 706, 740, 560, 24, '#FFFFFF');
      strokeRoundRect(ctx, 92, 706, 740, 560, 24, '#E7DAC8', 2);
      ctx.setFillStyle('#1F695D');
      ctx.setFontSize(28);
      ctx.fillText('最终排名', 134, 780);
      drawBoldText(ctx, '结算分值', 134, 834, '#1A1C1A', 48);

      let y = 918;
      rankingParticipants.forEach((p, index) => {
        const rank = index + 2;
        drawMedalBadge(ctx, rank, 172, y - 8);
        ctx.setStrokeStyle('#D8D8D8');
        ctx.setLineWidth(2);
        ctx.beginPath();
        ctx.moveTo(256, y - 42);
        ctx.lineTo(256, y + 28);
        ctx.stroke();
        drawBoldText(ctx, truncatePosterText(p.name, 6), 296, y + 10, '#1A1C1A', 46);
        ctx.setFillStyle(scoreColor(p.scoreTone));
        ctx.setFontSize(64);
        ctx.setTextAlign('right');
        ctx.fillText(p.score, 782, y + 12);
        ctx.setTextAlign('left');
        if (index < rankingParticipants.length - 1) {
          ctx.setStrokeStyle('#EDE3D4');
          ctx.setLineWidth(2);
          drawDashedLine(ctx, 134, y + 76, 790, 10, 8);
        }
        y += 132;
      });

      fillRoundRect(ctx, 132, 1360, 270, 270, 14, '#FFFFFF');
      strokeRoundRect(ctx, 132, 1360, 270, 270, 14, '#D8CDBA', 2);
      ctx.setStrokeStyle('#00604F');
      ctx.setLineWidth(2);
      drawDashedLine(ctx, 154, 1384, 380, 8, 6);
      drawVerticalDashedLine(ctx, 154, 1384, 1610, 8, 6);
      drawVerticalDashedLine(ctx, 380, 1384, 1610, 8, 6);
      drawDashedLine(ctx, 154, 1610, 380, 8, 6);
      ctx.drawImage(codePath, 170, 1408, 194, 194);

      ctx.setStrokeStyle('#E9DDCA');
      ctx.setLineWidth(2);
      drawVerticalDashedLine(ctx, 444, 1368, 1624, 8, 8);

      drawBoldText(ctx, '扫码查看公开结算页', 494, 1458, '#1A1C1A', 34);
      fillRoundRect(ctx, 494, 1500, 54, 7, 4, '#C9832B');
      ctx.setFillStyle('#6F7976');
      ctx.setFontSize(28);
      ctx.fillText('不含头像和计分明细', 494, 1572);

      ctx.setStrokeStyle('#E6D8C2');
      ctx.setLineWidth(2);
      ctx.beginPath();
      ctx.moveTo(100, 1688);
      ctx.lineTo(420, 1688);
      ctx.moveTo(506, 1688);
      ctx.lineTo(826, 1688);
      ctx.stroke();
      fillCircle(ctx, 464, 1688, 22, '#E9DDCA');
      ctx.setFillStyle('#FFFFFF');
      ctx.setFontSize(34);
      ctx.setTextAlign('center');
      ctx.fillText('1', 464, 1700);
      ctx.setTextAlign('left');

      ctx.draw(false, () => {
        wx.canvasToTempFilePath({
          canvasId: 'settlementPoster',
          width,
          height,
          destWidth: width,
          destHeight: height,
          fileType: 'png',
          success: (res) => resolve(res.tempFilePath),
          fail: reject,
        }, this);
      });
    });
  },

  saveImageToAlbum(filePath: string): Promise<void> {
    return new Promise((resolve, reject) => {
      wx.saveImageToPhotosAlbum({
        filePath,
        success: () => resolve(),
        fail: reject,
      });
    });
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
    if (isFinished && this.data.game.publicShareToken) {
      return {
        title: `【一局一分】牌局结算：“${this.data.game.name}”`,
        path: `/pages/game-detail/index?shareToken=${this.data.game.publicShareToken}`,
      };
    }
    return {
      title: '一局一分：邀请你加入牌局',
      path: `/pages/game-join/index?inviteCode=${this.data.inviteCode}`,
    };
  },
});
