import { createGame } from '../../services/game.service';
import { saveRecentSession } from '../../utils/storage';

Page({
  data: {
    name: '家庭聚会',
    nameLength: 4,
    zeroSumRequired: true,
  },
  onNameInput(event: WechatMiniprogram.Input) {
    const name = event.detail.value;
    this.setData({ name, nameLength: name.length });
  },
  onZeroSumChange(event: WechatMiniprogram.SwitchChange) {
    this.setData({ zeroSumRequired: event.detail.value });
  },
  async submit() {
    const name = String(this.data.name).trim();
    if (!name) return wx.showToast({ title: '请输入牌局名称', icon: 'none' });
    const game = await createGame(name, Boolean(this.data.zeroSumRequired));
    saveRecentSession(game.id);
    wx.redirectTo({ url: `/pages/game-detail/index?id=${game.id}` });
  },
});
