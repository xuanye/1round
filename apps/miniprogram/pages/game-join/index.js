"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
const auth_service_1 = require("../../services/auth.service");
const game_service_1 = require("../../services/game.service");
const storage_1 = require("../../utils/storage");
Page({
    data: {
        icons: {
            dice: '\uf522',
            close: '\uf00d',
            users: '\uf0c0',
            edit: '\uf304',
            info: '\uf05a',
            check: '\uf058',
            arrowRight: '\uf061',
        },
        inviteCode: '',
        game: null,
        participants: [],
        displayName: '',
        joining: false,
        joined: false,
        // Conflict game state
        conflictGame: null,
    },
    async onLoad(query) {
        let inviteCode = query.inviteCode || '';
        if (query.scene) {
            const scene = decodeURIComponent(query.scene);
            const match = scene.match(/(?:code=)?([A-Za-z0-9]+)/);
            if (match) {
                inviteCode = match[1];
            }
        }
        if (!inviteCode) {
            wx.showToast({ title: '邀请码缺失', icon: 'none' });
            setTimeout(() => wx.redirectTo({ url: '/pages/home/index' }), 1500);
            return;
        }
        this.setData({ inviteCode });
        wx.showLoading({ title: '正在获取预览...' });
        try {
            await (0, auth_service_1.ensureLogin)();
            // Check join preview
            const preview = await (0, game_service_1.joinPreview)(inviteCode);
            if (preview.alreadyJoined) {
                // Redirection for users already joined
                wx.redirectTo({ url: `/pages/game-detail/index?id=${preview.gameSessionId}&inviteCode=${inviteCode}` });
                return;
            }
            // Check if user has another active game
            const current = await (0, game_service_1.getCurrentGame)();
            if (current && current.id !== preview.gameSessionId) {
                const summary = await (0, game_service_1.getSummary)(current.id);
                const user = (0, storage_1.getUser)();
                let myScore = 0;
                const myPlayer = summary.players.find(p => p.userId === (user === null || user === void 0 ? void 0 : user.id) || (p.displayName === (user === null || user === void 0 ? void 0 : user.displayName) && p.userId === (user === null || user === void 0 ? void 0 : user.id)));
                if (myPlayer)
                    myScore = myPlayer.totalScore;
                this.setData({
                    conflictGame: {
                        id: current.id,
                        name: current.name,
                        canLeave: myScore === 0,
                    },
                });
            }
            const ownerInit = preview.ownerDisplayName ? preview.ownerDisplayName.slice(0, 1) : '创';
            this.setData({
                game: {
                    id: preview.gameSessionId,
                    name: preview.name,
                    creatorName: preview.ownerDisplayName,
                    creatorInitial: ownerInit,
                    participantText: `${preview.participantCount}${preview.maxParticipants ? ' / ' + preview.maxParticipants : ''}`,
                },
                participants: preview.participants.map(p => ({
                    id: p.id,
                    initial: p.displayName.slice(0, 1),
                })),
                displayName: preview.currentUserDisplayName,
            });
        }
        catch (err) {
            wx.showModal({
                title: '加入失败',
                content: err.message || '获取牌局预览失败',
                showCancel: false,
                success: () => wx.redirectTo({ url: '/pages/home/index' }),
            });
        }
        finally {
            wx.hideLoading();
        }
    },
    onNameInput(event) {
        this.setData({ displayName: String(event.detail.value) });
    },
    cancel() {
        wx.redirectTo({ url: '/pages/home/index' });
    },
    async submit() {
        const displayName = String(this.data.displayName).trim();
        if (!displayName) {
            wx.showToast({ title: '请输入显示名称', icon: 'none' });
            return;
        }
        this.setData({ joining: true });
        wx.showLoading({ title: '正在加入牌局...' });
        try {
            // If there is a conflict game, we must leave it first (and we only get here if canLeave is true)
            if (this.data.conflictGame) {
                if (!this.data.conflictGame.canLeave) {
                    throw new Error('原当前牌局分值不为 0，不能退出并加入新牌局');
                }
                await (0, game_service_1.leaveGame)(this.data.conflictGame.id);
            }
            const res = await (0, game_service_1.joinGame)(this.data.inviteCode, displayName);
            (0, storage_1.saveRecentSession)(res.gameSessionId); // Save recent session!
            this.setData({ joining: false, joined: true });
            wx.redirectTo({ url: `/pages/game-detail/index?id=${res.gameSessionId}&inviteCode=${this.data.inviteCode}` });
        }
        catch (err) {
            this.setData({ joining: false });
            wx.showToast({ title: err.message || '加入失败', icon: 'none' });
        }
        finally {
            wx.hideLoading();
        }
    },
});
//# sourceMappingURL=index.js.map