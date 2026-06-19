"use strict";
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
        id: 'mock-game-001',
        game: {
            name: '周末欢乐局',
            createdAt: '2024.05.20 14:30',
            inviteCode: 'A8K21P',
            isCreator: true,
        },
        participants: [
            { id: 'p1', initial: '明', name: '小明', role: '创建者', score: '+120', scoreTone: 'positive', isCreator: true },
            { id: 'p2', initial: '红', name: '小红', role: '已加入', score: '-50', scoreTone: 'negative', isCreator: false },
            { id: 'p3', initial: '华', name: '大华', role: '已加入', score: '-70', scoreTone: 'muted', isCreator: false },
        ],
        transfers: [
            { id: 't1', iconCode: '\uf362', from: '小明', to: '大华', multi: false, score: '+20', time: '10:45 AM' },
            { id: 't2', iconCode: '\uf0c0', from: '小红', to: '小明、大华', multi: true, score: '+50', time: '10:30 AM' },
            { id: 't3', iconCode: '\uf362', from: '大华', to: '小红', multi: false, score: '+15', time: '10:15 AM' },
        ],
    },
    onLoad(query) {
        this.setData({ id: query.id || this.data.id });
    },
    backHome() {
        wx.navigateBack({ fail: () => wx.redirectTo({ url: '/pages/home/index' }) });
    },
    showInvite() {
        wx.showModal({
            title: '加入二维码',
            content: `演示邀请码：${this.data.game.inviteCode}`,
            showCancel: false,
        });
    },
    inputScore() {
        wx.navigateTo({ url: `/pages/score-input/index?id=${this.data.id}` });
    },
    ranking() {
        wx.navigateTo({ url: `/pages/ranking/index?id=${this.data.id}` });
    },
    finish() {
        wx.showModal({
            title: '结束本局',
            content: '确认后将进入结算页。当前仅为页面演示，不会调用服务端。',
            confirmText: '确认结束',
            confirmColor: '#ba1a1a',
        });
    },
});
//# sourceMappingURL=index.js.map