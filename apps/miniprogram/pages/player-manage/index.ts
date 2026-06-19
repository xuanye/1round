import { addPlayer } from '../../services/game.service';

Page({
  data: { id: '', displayName: '' },
  onLoad(query: Record<string, string>) {
    this.setData({ id: query.id });
  },
  onInput(event: WechatMiniprogram.Input) {
    this.setData({ displayName: event.detail.value });
  },
  async submit() {
    const displayName = String(this.data.displayName).trim();
    if (!displayName) return wx.showToast({ title: '请输入玩家名称', icon: 'none' });
    await addPlayer(this.data.id, displayName);
    wx.navigateBack();
  },
});
