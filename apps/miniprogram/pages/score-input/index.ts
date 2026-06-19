type Receiver = {
  id: string;
  displayName: string;
  currentScore: number;
  initial: string;
  selected: boolean;
};

Page({
  data: {
    id: 'mock-game-001',
    receivers: [
      { id: 'p1', displayName: '陈晓明', currentScore: 1250, initial: '陈', selected: false },
      { id: 'p2', displayName: '林雨萌', currentScore: 890, initial: '林', selected: false },
      { id: 'p3', displayName: '赵刚', currentScore: 2100, initial: '赵', selected: false },
    ] as Receiver[],
    scoreText: '0',
    selectedCount: 0,
    canSubmit: false,
    submitText: '请选择接收方',
    deductionText: '你将扣除 0 分',
    allSelected: false,
    numKeys: ['1', '2', '3', '4', '5', '6', '7', '8', '9'],
  },
  onLoad(query: Record<string, string>) {
    this.setData({ id: query.id || this.data.id });
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
      allSelected: selectedCount === receivers.length,
    });
  },
  submit() {
    if (!this.data.canSubmit) return;
    wx.showToast({ title: '已记录页面演示', icon: 'success' });
    setTimeout(() => wx.navigateBack(), 450);
  },
});
