"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
const game_service_1 = require("../../services/game.service");
const storage_1 = require("../../utils/storage");
Page({
    data: {
        name: '家庭聚会',
        nameLength: 4,
        pickerRange: ['不限制', '2 人', '3 人', '4 人', '5 人', '6 人', '7 人', '8 人', '9 人', '10 人'],
        pickerIndex: 0,
    },
    onNameInput(event) {
        const name = event.detail.value;
        this.setData({ name, nameLength: name.length });
    },
    onMaxParticipantsChange(event) {
        this.setData({ pickerIndex: Number(event.detail.value) });
    },
    async submit() {
        const name = String(this.data.name).trim();
        if (!name)
            return wx.showToast({ title: '请输入牌局名称', icon: 'none' });
        // Derive maxParticipants
        const idx = this.data.pickerIndex;
        const maxParticipants = idx === 0 ? null : idx + 1;
        try {
            const game = await (0, game_service_1.createGame)(name, maxParticipants);
            (0, storage_1.saveRecentSession)(game.id);
            wx.redirectTo({ url: `/pages/game-detail/index?id=${game.id}&inviteCode=${game.inviteCode}` });
        }
        catch (err) {
            wx.showToast({ title: err.message || '创建失败', icon: 'none' });
        }
    },
});
//# sourceMappingURL=index.js.map