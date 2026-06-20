import { requireLogin } from '../../services/auth.service';
import { getSummary } from '../../services/game.service';
import { submitScoreTransfer } from '../../services/score.service';
import { getUser } from '../../utils/storage';

type Receiver = {
  id: string;
  displayName: string;
  currentScore: number;
  initial: string;
  selected: boolean;
};

function buildSubmitText(receivers: Receiver[], score: number): string {
  const selected = receivers.filter((receiver) => receiver.selected);
  if (selected.length === 0) return '请选择接收方';
  if (!Number.isInteger(score) || score <= 0) return '请输入分值';
  if (selected.length === 1) return `给 ${selected[0].displayName} +${score}`;
  return `给 ${selected.length} 人各 +${score}`;
}

function resolveSubmitErrorMessage(message: string): { feedback: string; navigateBack: boolean } {
  if (message.includes('已结算') || message.includes('未结算')) {
    return { feedback: '牌局状态已变化，请返回后刷新', navigateBack: true };
  }
  if (message.includes('参与者') || message.includes('退出')) {
    return { feedback: '参与者状态已变化，请返回后刷新', navigateBack: true };
  }
  return { feedback: message || '记分失败，请重试', navigateBack: false };
}

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
    helperText: '先选接收方，再输入每人分值',
    allSelected: false,
    numKeys: ['1', '2', '3', '4', '5', '6', '7', '8', '9'],
    submitting: false,
    feedbackMessage: '',
    feedbackTone: 'info' as 'info' | 'success' | 'error',
  },

  async onLoad(query: Record<string, string>) {
    const id = query.id || '';
    this.setData({ id });

    wx.showLoading({ title: '加载中...' });
    try {
      await requireLogin();
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
      this.applyState(receivers, '10');
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

  quickScore(event: WechatMiniprogram.TouchEvent) {
    const value = String(event.currentTarget.dataset.value);
    this.applyState(this.data.receivers, value);
  },

  applyState(receivers: Receiver[], scoreText: string, options?: { preserveFeedback?: boolean }) {
    const selectedCount = receivers.filter((receiver) => receiver.selected).length;
    const score = Number(scoreText);
    const canSubmit = selectedCount > 0 && Number.isInteger(score) && score > 0;
    const submitText = buildSubmitText(receivers, score);
    const helperText = canSubmit
      ? `本次你将扣除 ${score * selectedCount} 分`
      : selectedCount === 0
        ? '先选接收方，再输入每人分值'
        : '分值必须是大于 0 的整数';
    this.setData({
      receivers,
      scoreText,
      selectedCount,
      canSubmit,
      submitText,
      deductionText: `你将扣除 ${score * selectedCount} 分`,
      helperText,
      allSelected: receivers.length > 0 && selectedCount === receivers.length,
      feedbackMessage: options?.preserveFeedback ? this.data.feedbackMessage : '',
      feedbackTone: options?.preserveFeedback ? this.data.feedbackTone : 'info',
    });
  },

  async submit() {
    if (!this.data.canSubmit || this.data.submitting) return;

    this.setData({
      submitting: true,
      feedbackMessage: '正在记录分值...',
      feedbackTone: 'info',
      submitText: '正在提交...',
    });
    wx.showLoading({ title: '正在提交分值...' });
    const selectedIds = this.data.receivers
      .filter((r) => r.selected)
      .map((r) => r.id);
    const amount = Number(this.data.scoreText);
    const idempotencyKey = `score_transfer_${this.data.id}_${Date.now()}_${Math.random().toString(36).substring(2, 9)}`;

    try {
      await requireLogin();
      await submitScoreTransfer(this.data.id, selectedIds, amount, idempotencyKey);
      this.setData({
        feedbackMessage: '分值已记录，正在返回牌局',
        feedbackTone: 'success',
        submitText: '已记录',
      });
      wx.showToast({ title: '分值已记录', icon: 'success' });
      setTimeout(() => wx.navigateBack(), 600);
    } catch (err) {
      const resolved = resolveSubmitErrorMessage((err as any).message || '');
      this.applyState(this.data.receivers, this.data.scoreText, { preserveFeedback: true });
      this.setData({
        submitting: false,
        feedbackMessage: resolved.feedback,
        feedbackTone: 'error',
      });
      wx.showToast({ title: resolved.feedback, icon: 'none' });
      if (resolved.navigateBack) {
        setTimeout(() => this.goBack(), 700);
      }
    } finally {
      wx.hideLoading();
    }
  },
});
