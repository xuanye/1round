"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
const game_service_1 = require("../../services/game.service");
const storage_1 = require("../../utils/storage");
Page({
    data: {
        name: '家庭聚会',
        nameLength: 4,
        zeroSumRequired: true,
    },
    onNameInput(event) {
        const name = event.detail.value;
        this.setData({ name, nameLength: name.length });
    },
    onZeroSumChange(event) {
        this.setData({ zeroSumRequired: event.detail.value });
    },
    async submit() {
        const name = String(this.data.name).trim();
        if (!name)
            return wx.showToast({ title: '请输入牌局名称', icon: 'none' });
        const game = await (0, game_service_1.createGame)(name, Boolean(this.data.zeroSumRequired));
        (0, storage_1.saveRecentSession)(game.id);
        wx.redirectTo({ url: `/pages/game-detail/index?id=${game.id}` });
    },
});
//# sourceMappingURL=index.js.map