"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
const game_service_1 = require("../../services/game.service");
Page({
    data: { id: '', displayName: '' },
    onLoad(query) {
        this.setData({ id: query.id });
    },
    onInput(event) {
        this.setData({ displayName: event.detail.value });
    },
    async submit() {
        const displayName = String(this.data.displayName).trim();
        if (!displayName)
            return wx.showToast({ title: '请输入玩家名称', icon: 'none' });
        await (0, game_service_1.addPlayer)(this.data.id, displayName);
        wx.navigateBack();
    },
});
//# sourceMappingURL=index.js.map