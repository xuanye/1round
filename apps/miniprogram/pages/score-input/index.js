"use strict";
Page({
    data: {
        id: 'mock-game-001',
        receivers: [
            { id: 'p1', displayName: '陈晓明', currentScore: 1250, initial: '陈', selected: false, avatarUrl: 'https://lh3.googleusercontent.com/aida-public/AB6AXuDektPOpIegFaoK9by-AsDJt7SHjJcOmLqjox3GQBDEgN6QdBLc73dzPRxvV4bCj15TFGiCEXsxXzifTYfdTfQqCJt7141EXnzBnFlCtsh9brM9ka_ct-KxCNZmgvnaWYrTfBb06iUapRX3r3s1utDfICg6vNZ96X1Suc0J9OAWDeXyg0cTWSkccacetudc0iB-DyE1iPsVSQcIO-sREsjZrF2xuV4RHZd-P7Ku7SVisJnfmRvttANrRg' },
            { id: 'p2', displayName: '林雨萌', currentScore: 890, initial: '林', selected: false, avatarUrl: 'https://lh3.googleusercontent.com/aida-public/AB6AXvABv5NYK2D_VYIXy6Vn5h2af5iqEuAk-czqTpQ-MW1d7sHHws7uo-D6gD0l-pM9tGUaI_q-LFx671cuYNJyuPe1YL5UydFEwwgwrZNuHT1IxCCh1RwNeWCOCaeS2x_kNPlzlmnE9zZbtfb_FJHOu9RXKYqmk4DwI-ccpjheHo-8hWWTB-QxdNq0QSdfz4N9Fp3f58inRYCva1Ca5FIse73bbnJ0EectQBGfINrGOQeR9Hx9hwjCtazodg' },
            { id: 'p3', displayName: '赵刚', currentScore: 2100, initial: '赵', selected: false, avatarUrl: 'https://lh3.googleusercontent.com/aida-public/AB6AXuA9QA-q1wJKZrHMHD1ePyWtiUtEClsMAkRAfmf-JZncqSxz2NCeLckjXVucMi6dbuOH_ea0KyV5D4QTCWSPwKKNs6YnrbRyWd1dwiHga4RP4GYBjg4v2KNKzZwAd0Y32UetmtxJ5EyZhBJu0yjuvUnjRxrEXmVWs9VpQ3oBzbqkDnrFoNRrCCAe4X_ZZl_-G1UD9jIiNWSejFFZjkNhNM9ueh5nn6OoOwhds_MIdYYs7JpBB2wzC6HULg' },
        ],
        scoreText: '0',
        selectedCount: 0,
        canSubmit: false,
        submitText: '请选择接收方',
        deductionText: '你将扣除 0 分',
        allSelected: false,
        numKeys: ['1', '2', '3', '4', '5', '6', '7', '8', '9'],
    },
    onLoad(query) {
        this.setData({ id: query.id || this.data.id });
    },
    goBack() {
        wx.navigateBack({ fail: () => wx.redirectTo({ url: `/pages/game-detail/index?id=${this.data.id}` }) });
    },
    toggleReceiver(event) {
        const id = String(event.currentTarget.dataset.id);
        const receivers = this.data.receivers.map((receiver) => (receiver.id === id ? { ...receiver, selected: !receiver.selected } : receiver));
        this.applyState(receivers, this.data.scoreText);
    },
    toggleAll() {
        const nextSelected = !this.data.allSelected;
        const receivers = this.data.receivers.map((receiver) => ({ ...receiver, selected: nextSelected }));
        this.applyState(receivers, this.data.scoreText);
    },
    pressKey(event) {
        const value = String(event.currentTarget.dataset.value);
        let scoreText = this.data.scoreText;
        if (value === 'clear') {
            scoreText = scoreText.length > 1 ? scoreText.slice(0, -1) : '0';
        }
        else if (scoreText === '0') {
            scoreText = value === '0' ? '0' : value;
        }
        else if (scoreText.length < 5) {
            scoreText += value;
        }
        this.applyState(this.data.receivers, scoreText);
    },
    applyState(receivers, scoreText) {
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
        if (!this.data.canSubmit)
            return;
        wx.showToast({ title: '已记录页面演示', icon: 'success' });
        setTimeout(() => wx.navigateBack(), 450);
    },
});
//# sourceMappingURL=index.js.map