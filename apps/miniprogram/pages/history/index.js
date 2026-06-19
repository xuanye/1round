"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
const storage_1 = require("../../utils/storage");
Page({
    data: { ids: [] },
    onShow() {
        this.setData({ ids: (0, storage_1.getRecentSessions)() });
    },
    open(event) {
        wx.navigateTo({ url: `/pages/game-detail/index?id=${event.currentTarget.dataset.id}` });
    },
});
//# sourceMappingURL=index.js.map