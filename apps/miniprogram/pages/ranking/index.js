"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
const auth_service_1 = require("../../services/auth.service");
const game_service_1 = require("../../services/game.service");
const format_1 = require("../../utils/format");
Page({
    data: {
        id: '',
        players: []
    },
    async onLoad(query) {
        const id = query.id || '';
        this.setData({ id });
        wx.showLoading({ title: '加载中...' });
        try {
            await (0, auth_service_1.ensureLogin)();
            const list = await (0, game_service_1.getRanking)(id);
            this.setData({
                players: list.map((item) => ({
                    ...item,
                    initial: item.displayName.slice(0, 1),
                    scoreLabel: (0, format_1.formatScore)(item.totalScore),
                    scoreTone: item.totalScore > 0 ? 'positive' : item.totalScore < 0 ? 'negative' : 'muted',
                    averageLabel: String(item.averageScore || 0),
                })),
            });
        }
        catch (err) {
            wx.showToast({ title: err.message || '获取排行榜失败', icon: 'none' });
        }
        finally {
            wx.hideLoading();
        }
    },
});
//# sourceMappingURL=index.js.map