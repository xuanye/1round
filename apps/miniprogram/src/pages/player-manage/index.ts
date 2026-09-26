import { requireLogin } from '../../services/auth.service';
import { updateMyProfile } from '../../services/game.service';
import { getSystemFontClass } from '../../utils/system-font';

const NAME_MAX_LENGTH = 12;

Page({
  data: {
    fontClass: getSystemFontClass(),
    id: '',
    displayName: '',
    nameLength: 0,
    nameFocused: false,
    submitting: false,
  },
  async onLoad(query: Record<string, string>) {
    try {
      await requireLogin();
    } catch (err) {
      wx.showToast({ title: (err as any).message || '登录失败', icon: 'none' });
      wx.switchTab({ url: '/pages/home/index' });
      return;
    }

    const id = query.id || '';
    const displayName = query.displayName ? decodeURIComponent(query.displayName) : '';
    this.setData({ id, displayName, nameLength: displayName.length });
  },
  onInput(event: WechatMiniprogram.Input) {
    const displayName = event.detail.value;
    this.setData({ displayName, nameLength: displayName.length });
  },
  onNameFocus() {
    this.setData({ nameFocused: true });
  },
  onNameBlur() {
    this.setData({ nameFocused: false });
  },
  clearName() {
    this.setData({ displayName: '', nameLength: 0, nameFocused: true });
  },
  async submit() {
    if (this.data.submitting) return;

    const displayName = String(this.data.displayName).trim();
    if (!displayName) return wx.showToast({ title: '请输入玩家名称', icon: 'none' });
    if (displayName.length > NAME_MAX_LENGTH) {
      return wx.showToast({ title: `名称最多${NAME_MAX_LENGTH}字`, icon: 'none' });
    }

    this.setData({ submitting: true });
    try {
      await requireLogin();
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
      this.setData({ submitting: false });
    }
  },
});
