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
      { id: 'p1', initial: '周', avatarUrl: 'https://lh3.googleusercontent.com/aida-public/AB6AXuCnx3j19hP4nb-3iIf88qb3AP11TRrIJJlvC0wSd30_mQ4nBpbk1Y2aJPl4rG0gRhfZ2YQCfWABFJlESgNggKmdG4CYXD4byY4xRXpvu1Is5ubv9yziVhks7byd6sMfKtuMWJPbxSsV0uUzc9HQnnHDldoI5aH9V1b3gRzRoKcWNtEeo-sMlgENpiqiWIMsFmbApYIN2n-Wy9mb9FVxFeevVFyWU2IOJE4Iux8eM8-f0BWewd6jjoTKLA' },
      { id: 'p2', initial: '林', avatarUrl: 'https://lh3.googleusercontent.com/aida-public/AB6AXuCbUuyQ9d_GacZDB29CmMfSJ8UUUpxvpe-kqil5623re1Jao0GqPfjKAxw1VyPtBScfxvjJaWWdVNCwvZ1RgXMlGUQznDvp0VQBf6As5MOnq0WN71H1JwthDRpd04UCcsxqxSwUeK4zysUm169l7QwQOglXe12KyTa-lTNTchBXRDQZc7r9e4mCK6BkybWua-6ziEt6gWUEBMt7MrizGxqLmg_wb_fP649OuWdE7VgbEQdhIVN3qGG6yQ' },
      { id: 'p3', initial: '赵', avatarUrl: 'https://lh3.googleusercontent.com/aida-public/AB6AXuD9kWMBbwc3Qe5TJj2fgLThcBXwPVWsp7S6SnqaynLhU1d1RiyRiDgzHWmlfRzpn5XOps3f_QR2X44g5TWTn9FqHLLmGqCrG92EhkGEGf7IFnGCY2qjRQxu8azP1mRxwyrbB9nkXU6Z9dPNoOk_0pITv_am8cWZuTX4-HVV70ODlomMIukbfkpYRSwMq0u6T7VXuoNV9hYv20aoq423b8I9gvbp9iqfX_P7XJq95wIkfXcNydW5SEFaKg' },
      { id: 'p4', initial: '+1' },
    ],
    displayName: '老书记05',
    joining: false,
    joined: false,
  },
  onNameInput(event: WechatMiniprogram.Input) {
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
