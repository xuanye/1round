import { requireLogin } from '../../services/auth.service';
import { createGame } from '../../services/game.service';
import { saveRecentSession } from '../../utils/storage';

Page({
  data: {
    name: '家庭聚会',
    nameLength: 4,
    pickerRange: ['不限制', '2 人', '3 人', '4 人', '5 人', '6 人', '7 人', '8 人', '9 人', '10 人'],
    pickerIndex: 0,
  },
  async onLoad() {
    try {
      await requireLogin();
    } catch (err) {
      wx.showToast({ title: (err as any).message || '登录失败', icon: 'none' });
      wx.redirectTo({ url: '/pages/home/index' });
    }
  },
  onNameInput(event: WechatMiniprogram.Input) {
    const name = event.detail.value;
    this.setData({ name, nameLength: name.length });
  },
  onMaxParticipantsChange(event: WechatMiniprogram.PickerChange) {
    this.setData({ pickerIndex: Number(event.detail.value) });
  },
  async submit() {
    await requireLogin();

    const name = String(this.data.name).trim();
    if (!name) return wx.showToast({ title: '请输入牌局名称', icon: 'none' });
    
    // Derive maxParticipants
    const idx = this.data.pickerIndex;
    const maxParticipants = idx === 0 ? null : idx + 1;

    try {
      const game = await createGame(name, maxParticipants);
      saveRecentSession(game.id);
      wx.redirectTo({ url: `/pages/game-detail/index?id=${game.id}&inviteCode=${game.inviteCode}` });
    } catch (err) {
      wx.showToast({ title: (err as any).message || '创建失败', icon: 'none' });
    }
  },
});
