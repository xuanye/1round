"use strict";
Page({
    data: {
        game: {
            id: 'mock-game-001',
            name: '周末家庭聚会 (2024-05-20)',
            creatorName: '老书记01',
            creatorInitial: '老',
            participantText: '4 / 8',
        },
        participants: [
            { id: 'p1', initial: '周' },
            { id: 'p2', initial: '林' },
            { id: 'p3', initial: '赵' },
            { id: 'p4', initial: '+1' },
        ],
        displayName: '老书记05',
        joining: false,
        joined: false,
    },
    onNameInput(event) {
        this.setData({ displayName: String(event.detail.value) });
    },
    cancel() {
        wx.navigateBack();
    },
    submit() {
        if (!String(this.data.displayName).trim()) {
            wx.showToast({ title: '请输入显示名称', icon: 'none' });
            return;
        }
        this.setData({ joining: true });
        setTimeout(() => {
            this.setData({ joining: false, joined: true });
            wx.redirectTo({ url: `/pages/game-detail/index?id=${this.data.game.id}` });
        }, 450);
    },
});
//# sourceMappingURL=index.js.map