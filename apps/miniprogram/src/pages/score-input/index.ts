import { ensureLogin } from '../../services/auth.service';
import { getSummary } from '../../services/game.service';
import { submitScoreTransfer } from '../../services/score.service';
import { getUser } from '../../utils/storage';

type Receiver = {
  id: string;
  displayName: string;
  currentScore: number;
  initial: string;
  selected: boolean;
  avatarUrl?: string;
};

Page({
  data: {
    icons: {
      back: '\uf060',
      info: '\uf05a',
      check: '\uf058',
      deleteLeft: '\uf55a',
      doneAll: '\uf560',
      transfer: '\uf362',
    },
    id: '',
    receivers: [] as Receiver[],
    scoreText: '0',
    selectedCount: 0,
    canSubmit: false,
    submitText: '请选择接收方',
    deductionText: '你将扣除 0 分',
    allSelected: false,
    numKeys: ['1', '2', '3', '4', '5', '6', '7', '8', '9'],
    submitting: false,
  },

  async onLoad(query: Record<string, string>) {
    const id = query.id || '';
    this.setData({ id });

    wx.showLoading({ title: '加载中...' });
    try {
      await ensureLogin();
      const summary = await getSummary(id);
      const user = getUser();

      const receivers = summary.players
        .filter((p) => p.userId !== user?.id && p.displayName !== user?.displayName)
        .map((p) => ({
          id: p.id,
          displayName: p.displayName,
          currentScore: p.totalScore,
          initial: p.displayName.slice(0, 1),
          selected: false,
        }));

      this.setData({ receivers });
      this.applyState(receivers, '0');
    } catch (err) {
      wx.showToast({ title: (err as any).message || '获取成员失败', icon: 'none' });
    } finally {
      wx.hideLoading();
    }
  },

  goBack() {
    wx.navigateBack({ fail: () => wx.redirectTo({ url: `/pages/game-detail/index?id=${this.data.id}` }) });
  },

  toggleReceiver(event: WechatMiniprogram.TouchEvent) {
    const id = String(event.currentTarget.dataset.id);
    const receivers = this.data.receivers.map((receiver) => (
      receiver.id === id ? { ...receiver, selected: !receiver.selected } : receiver
    ));
    this.applyState(receivers, this.data.scoreText);
  },

  toggleAll() {
    const nextSelected = !this.data.allSelected;
    const receivers = this.data.receivers.map((receiver) => ({ ...receiver, selected: nextSelected }));
    this.applyState(receivers, this.data.scoreText);
  },

  pressKey(event: WechatMiniprogram.TouchEvent) {
    const value = String(event.currentTarget.dataset.value);
    let scoreText = this.data.scoreText;
    if (value === 'clear') {
      scoreText = scoreText.length > 1 ? scoreText.slice(0, -1) : '0';
    } else if (scoreText === '0') {
      scoreText = value === '0' ? '0' : value;
    } else if (scoreText.length < 5) {
      scoreText += value;
    }
    this.applyState(this.data.receivers, scoreText);
  },

  applyState(receivers: Receiver[], scoreText: string) {
    const selectedCount = receivers.filter((receiver) => receiver.selected).length;
    const score = Number(scoreText);
    const canSubmit = selectedCount > 0 && Number.isInteger(score) && score > 0;
    const submitText = canSubmit
      ? `给 ${selectedCount} 人各 +${score}`
      : selectedCount === 0
        ? '请选择接收方'
        : '请输入分值';
    this.setData({
      receivers,
      scoreText,
      selectedCount,
      canSubmit,
      submitText,
      deductionText: `你将扣除 ${score * selectedCount} 分`,
      allSelected: receivers.length > 0 && selectedCount === receivers.length,
    });
  },

  async submit() {
    if (!this.data.canSubmit || this.data.submitting) return;

    this.setData({ submitting: true });
    wx.showLoading({ title: '正在提交分值...' });
    const selectedIds = this.data.receivers
      .filter((r) => r.selected)
      .map((r) => r.id);
    const amount = Number(this.data.scoreText);
    const idempotencyKey = `score_transfer_${this.data.id}_${Date.now()}_${Math.random().toString(36).substring(2, 9)}`;

    try {
      await submitScoreTransfer(this.data.id, selectedIds, amount, idempotencyKey);
      wx.showToast({ title: '分值已记录', icon: 'success' });
      setTimeout(() => wx.navigateBack(), 600);
    } catch (err) {
      this.setData({ submitting: false });
      wx.showToast({ title: (err as any).message || '记分失败', icon: 'none' });
    } finally {
      wx.hideLoading();
    }
  },
});
