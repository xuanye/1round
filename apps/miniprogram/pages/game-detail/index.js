"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
const auth_service_1 = require("../../services/auth.service");
const game_service_1 = require("../../services/game.service");
const storage_1 = require("../../utils/storage");
const realtime_service_1 = require("../../services/realtime.service");
const qrcode_1 = require("../../utils/qrcode");
const format_1 = require("../../utils/format");
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
            status: 'active',
            publicShareToken: '',
        },
        participants: [],
        transfers: [],
        pendingFinishRequest: null,
        // Invite Overlay
        showInviteOverlay: false,
        qrCodeUrl: '',
        // Pagination
        hasMoreTransfers: true,
        isLoadingTransfers: false,
    },
    realtime: null,
    async onLoad(query) {
        const id = query.id || '';
        const inviteCode = query.inviteCode || '';
        const shareToken = query.shareToken || '';
        this.setData({ id, inviteCode, shareToken });
        if (id) {
            this.realtime = new realtime_service_1.RealtimeService();
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
            await (0, auth_service_1.ensureLogin)();
            await this.loadGameData();
            // Connect to websocket if the game is active
            if (this.data.game.status === 'active' && this.realtime) {
                this.realtime.connect(this.data.id);
                this.realtime.onEvent(() => {
                    // Re-fetch everything on socket notifications
                    this.loadGameData();
                });
            }
        }
        catch (err) {
            console.error('Game detail load failed:', err);
            wx.showToast({ title: err.message || '加载失败', icon: 'none' });
        }
        finally {
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
            const user = (0, storage_1.getUser)();
            const summary = await (0, game_service_1.getSummary)(id);
            (0, storage_1.saveRecentSession)(id);
            // Determine creator using robust ownerUserId field
            const isCreator = summary.ownerUserId === (user === null || user === void 0 ? void 0 : user.id);
            const isFinished = summary.status === 'finished';
            if (isFinished) {
                if (this.realtime)
                    this.realtime.disconnect();
                await this.loadSettledGame();
                return;
            }
            const participants = summary.players.map((p) => {
                const isMe = p.userId === (user === null || user === void 0 ? void 0 : user.id) || (p.displayName === (user === null || user === void 0 ? void 0 : user.displayName) && p.userId === (user === null || user === void 0 ? void 0 : user.id));
                const isPlayerCreator = p.userId === summary.ownerUserId;
                return {
                    id: p.id,
                    initial: p.displayName.slice(0, 1),
                    name: p.displayName,
                    role: isPlayerCreator ? '创建者' : '已加入',
                    score: (0, format_1.formatScore)(p.totalScore),
                    scoreTone: p.totalScore > 0 ? 'positive' : p.totalScore < 0 ? 'negative' : 'muted',
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
        }
        catch (err) {
            console.error('Fetch game data failed:', err);
        }
    },
    async loadTransfers(reload = false) {
        if (this.data.isLoadingTransfers)
            return;
        if (!reload && !this.data.hasMoreTransfers)
            return;
        this.setData({ isLoadingTransfers: true });
        let beforeSeq;
        if (!reload && this.data.transfers.length > 0) {
            beforeSeq = this.data.transfers[this.data.transfers.length - 1].sequenceNo;
        }
        try {
            const list = await (0, game_service_1.getScoreTransfers)(this.data.id, beforeSeq, 20);
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
        }
        catch (err) {
            console.error('Fetch transfers failed:', err);
            this.setData({ isLoadingTransfers: false });
        }
    },
    async loadSettledGame() {
        try {
            const detail = await (0, game_service_1.getSettlementDetail)(this.data.id);
            (0, storage_1.saveRecentSession)(this.data.id);
            const user = (0, storage_1.getUser)();
            const mappedParticipants = detail.participants.map(p => ({
                id: p.id,
                initial: p.displayName.slice(0, 1),
                name: p.displayName,
                role: '已结算',
                score: (0, format_1.formatScore)(p.finalScore),
                scoreTone: p.finalScore > 0 ? 'positive' : p.finalScore < 0 ? 'negative' : 'muted',
                isCreator: false,
                isMe: p.displayName === (user === null || user === void 0 ? void 0 : user.displayName),
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
        }
        catch (err) {
            console.error('Load settled game failed:', err);
        }
    },
    async loadPublicSettlement() {
        try {
            const detail = await (0, game_service_1.getPublicSettlement)(this.data.shareToken);
            const mappedParticipants = detail.participants.map(p => ({
                id: p.id,
                initial: p.displayName.slice(0, 1),
                name: p.displayName,
                role: '公开结算',
                score: `${p.finalScore >= 0 ? '+' : ''}${p.finalScore}`,
                scoreTone: p.finalScore > 0 ? 'positive' : p.finalScore < 0 ? 'negative' : 'muted',
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
        }
        catch (err) {
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
                (0, qrcode_1.drawQRCode)('inviteQR', inviteLink, 180, this);
            });
        });
    },
    hideInvite() {
        this.setData({ showInviteOverlay: false });
    },
    none() { },
    inputScore() {
        wx.navigateTo({ url: `/pages/score-input/index?id=${this.data.id}` });
    },
    ranking() {
        wx.navigateTo({ url: `/pages/ranking/index?id=${this.data.id}` });
    },
    renameSelf() {
        const me = this.data.participants.find(p => p.isMe);
        if (!me)
            return;
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
                            await (0, game_service_1.finishGameDirect)(self.data.id);
                            wx.showToast({ title: '牌局已结束', icon: 'success' });
                            self.loadGameData();
                        }
                        catch (err) {
                            wx.showToast({ title: err.message || '操作失败', icon: 'none' });
                        }
                    }
                },
            });
        }
        else {
            wx.showModal({
                title: '申请结束牌局',
                content: '确定要向创建者发起结束牌局的申请吗？',
                success: async (res) => {
                    if (res.confirm) {
                        try {
                            await (0, game_service_1.requestFinish)(self.data.id);
                            wx.showToast({ title: '已发起申请', icon: 'success' });
                            self.loadGameData();
                        }
                        catch (err) {
                            wx.showToast({ title: err.message || '发起失败', icon: 'none' });
                        }
                    }
                },
            });
        }
    },
    async approveFinish() {
        if (!this.data.pendingFinishRequest)
            return;
        try {
            await (0, game_service_1.approveFinishRequest)(this.data.id, this.data.pendingFinishRequest.id);
            wx.showToast({ title: '已同意结束', icon: 'success' });
            this.loadGameData();
        }
        catch (err) {
            wx.showToast({ title: err.message || '操作失败', icon: 'none' });
        }
    },
    async rejectFinish() {
        if (!this.data.pendingFinishRequest)
            return;
        try {
            await (0, game_service_1.rejectFinishRequest)(this.data.id, this.data.pendingFinishRequest.id);
            wx.showToast({ title: '已拒绝结束', icon: 'success' });
            this.loadGameData();
        }
        catch (err) {
            wx.showToast({ title: err.message || '操作失败', icon: 'none' });
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
//# sourceMappingURL=index.js.map