"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
const game_service_1 = require("../../services/game.service");
Page({
    data: { id: '', displayName: '' },
    onLoad(query) {
        const id = query.id || '';
        const displayName = query.displayName ? decodeURIComponent(query.displayName) : '';
        this.setData({ id, displayName });
    },
    onInput(event) {
        this.setData({ displayName: event.detail.value });
    },
    async submit() {
        const displayName = String(this.data.displayName).trim();
        if (!displayName)
            return wx.showToast({ title: '请输入玩家名称', icon: 'none' });
        try {
            await (0, game_service_1.updateMyProfile)(this.data.id, displayName);
            // Success, update local storage display name if needed
            const user = wx.getStorageSync('one_round_user');
            if (user) {
                user.displayName = displayName;
                wx.setStorageSync('one_round_user', user);
            }
            wx.navigateBack();
        }
        catch (err) {
            wx.showToast({ title: err.message || '修改失败', icon: 'none' });
        }
    },
});
//# sourceMappingURL=index.js.map