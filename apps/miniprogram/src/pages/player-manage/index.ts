import { requireLogin } from '../../services/auth.service';
import { updateMyProfile } from '../../services/game.service';

Page({
  data: { id: '', displayName: '' },
  async onLoad(query: Record<string, string>) {
    try {
      await requireLogin();
    } catch (err) {
      wx.showToast({ title: (err as any).message || '登录失败', icon: 'none' });
      wx.redirectTo({ url: '/pages/home/index' });
      return;
    }

    const id = query.id || '';
    const displayName = query.displayName ? decodeURIComponent(query.displayName) : '';
    this.setData({ id, displayName });
  },
  onInput(event: WechatMiniprogram.Input) {
    this.setData({ displayName: event.detail.value });
  },
  async submit() {
    try {
      await requireLogin();

      const displayName = String(this.data.displayName).trim();
      if (!displayName) return wx.showToast({ title: '请输入玩家名称', icon: 'none' });
      await updateMyProfile(this.data.id, displayName);
      // Success, update local storage display name if needed
      const user = wx.getStorageSync('one_round_user');
      if (user) {
        user.displayName = displayName;
        wx.setStorageSync('one_round_user', user);
      }
      wx.navigateBack();
    } catch (err) {
      wx.showToast({ title: (err as any).message || '修改失败', icon: 'none' });
    }
  },
});
