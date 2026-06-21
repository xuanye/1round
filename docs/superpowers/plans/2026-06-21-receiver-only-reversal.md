# Receiver-Only Score Reversal Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Allow only the receivers of a score transfer to reverse it, and implement the corresponding UI icon entrypoint on the WeChat Mini Program game detail page.

**Architecture:** Update backend Go transactional check to restrict score transfer reversals to receiver player IDs. Update frontend typescript model/type declarations, compute eligibility for the current player's ID, and render a subtle undo icon binding the reversal API.

**Tech Stack:** Go 1.24, chi, database/sql, TypeScript, WeChat Native Mini Program (WXML, WXSS)

## Global Constraints
- **Do not commit** unless explicitly instructed by the user.
- **SQLite constraints** are real product constraints; keep writes transactional.
- **No secrets in client**; domain and auth keys stay in backend.
- **Surgical changes**; only modify what is necessary.

---

### Task 1: Backend Reversal Authorization & Tests

**Files:**
- Modify: [service.go](file:///Users/xuanye/workspaces/1round/apps/server/internal/app/scoretransfer/service.go#L326-L330)
- Modify: [round_cycle_test.go](file:///Users/xuanye/workspaces/1round/apps/server/tests/round_cycle_test.go#L141-L147)

**Interfaces:**
- Consumes: `Queries.GetScoreTransferForUpdate` loading receivers.
- Produces: Updated authorization logic in `Service.Reverse` that returns `domain.ErrForbidden` when caller is not a receiver.

- [ ] **Step 1: Modify backend authorization check**
  Replace lines 326-330 in `apps/server/internal/app/scoretransfer/service.go`:
  ```go
  // Old check:
  // if original.FromPlayerID != callerPlayer.ID && session.OwnerUserID != userID {
  //     return domain.ErrForbidden
  // }
  ```
  with:
  ```go
  allowed := false
  for _, r := range original.Receivers {
      if r.PlayerID == callerPlayer.ID {
          allowed = true
          break
      }
  }
  if !allowed {
      return domain.ErrForbidden
  }
  ```

- [ ] **Step 2: Update integration tests**
  Modify the test case `TestReversalReopensRoundInsteadOfAdvancing` in `apps/server/tests/round_cycle_test.go` to call `Reverse` as the receiver `u2.ID` (player `p2`) instead of `owner.ID`:
  ```go
  // Old code:
  // if _, err := app.scoreTransfer.Reverse(ctx, owner.ID, game.ID, first.ID, scoretransfersvc.ReverseInput{
  ```
  with:
  ```go
  if _, err := app.scoreTransfer.Reverse(ctx, u2.ID, game.ID, first.ID, scoretransfersvc.ReverseInput{
  ```

- [ ] **Step 3: Run backend test suite**
  Run all tests in the backend package to verify correct behavior:
  Run: `cd /Users/xuanye/workspaces/1round/apps/server && go test ./...`
  Expected: All tests PASS.

---

### Task 2: Frontend Types and UI Icon Registration

**Files:**
- Modify: [score-transfer.ts](file:///Users/xuanye/workspaces/1round/apps/miniprogram/src/models/score-transfer.ts)
- Modify: [game-detail/index.ts](file:///Users/xuanye/workspaces/1round/apps/miniprogram/src/pages/game-detail/index.ts#L56-L89)

**Interfaces:**
- Consumes: Mapped fields from `getScoreTransfers` response.
- Produces: Updated model types `ScoreTransfer`, `DetailTransfer`, and registry for the `undo` icon code in the page.

- [ ] **Step 1: Update client model type**
  Modify `apps/miniprogram/src/models/score-transfer.ts` to include optional fields:
  ```typescript
  export type ScoreTransfer = {
    id: string;
    sequenceNo: number;
    fromPlayerId: string;
    receiverPlayerIds: string[];
    amount: number;
    createdAt: string;
    text: string;
    transferKind?: string;
    reversalOfTransferId?: string;
    reversedAt?: string;
  };
  ```

- [ ] **Step 2: Update local page DetailTransfer type**
  Modify the type definition `DetailTransfer` in `apps/miniprogram/src/pages/game-detail/index.ts`:
  ```typescript
  type DetailTransfer = {
    id: string;
    iconCode: string;
    text: string;
    time: string;
    sequenceNo: number;
    receiverPlayerIds: string[];
    transferKind?: string;
    reversalOfTransferId?: string;
    reversedAt?: string;
    canReverse: boolean;
  };
  ```

- [ ] **Step 3: Add undo icon code to icons list**
  In the `Page` constructor object data init in `apps/miniprogram/src/pages/game-detail/index.ts`:
  ```typescript
  icons: {
    back: '\uf060',
    qrCode: '\uf029',
    ranking: '\ue561',
    star: '\uf005',
    plusCircle: '\uf055',
    history: '\uf1da',
    flag: '\uf024',
    home: '\uf015',
    chart: '\uf201',
    info: '\uf05a',
    undo: '\uf0e2', // Added undo icon code
  },
  ```

- [ ] **Step 4: Register myPlayerId in page data**
  In the `data` object of `apps/miniprogram/src/pages/game-detail/index.ts`, add the `myPlayerId` key:
  ```typescript
  data: {
    // ... other properties
    uninvolvedNamesText: '',
    myPlayerId: '', // Added myPlayerId
  ```

---

### Task 3: Page Logic Mapping, Event Handling and WXML Template Rendering

**Files:**
- Modify: [game-detail/index.ts](file:///Users/xuanye/workspaces/1round/apps/miniprogram/src/pages/game-detail/index.ts#L185-L281)
- Modify: [game-detail/index.wxml](file:///Users/xuanye/workspaces/1round/apps/miniprogram/src/pages/game-detail/index.wxml#L111-L128)

**Interfaces:**
- Consumes: `reverseScoreTransfer` from `../../services/score.service`.
- Produces: Correct computation of `canReverse` per item, interactive popup confirmation modal, and UI rendering of undo icon.

- [ ] **Step 1: Extract myPlayerId in loadGameData**
  Modify `loadGameData` in `apps/miniprogram/src/pages/game-detail/index.ts` to lookup and save `myPlayerId`:
  ```typescript
  // Inside loadGameData:
  const participants = summary.players.map((p) => {
    // ...
  });
  
  const me = participants.find(p => p.isMe);
  const myPlayerId = me ? me.id : '';
  ```
  Set this to the page's data:
  ```typescript
  this.setData({
    // ...
    participants,
    myPlayerId,
    // ...
  });
  ```
  And call transfers loading with it:
  ```typescript
  // Load first page of transfers
  await this.loadTransfers(true, myPlayerId);
  ```

- [ ] **Step 2: Update loadTransfers to calculate canReverse**
  Modify `loadTransfers` signature and map function in `apps/miniprogram/src/pages/game-detail/index.ts`:
  ```typescript
  async loadTransfers(reload = false, currentMyPlayerId?: string) {
    if (this.data.isLoadingTransfers) return;
    if (!reload && !this.data.hasMoreTransfers) return;

    this.setData({ isLoadingTransfers: true });

    let beforeSeq: number | undefined;
    if (!reload && this.data.transfers.length > 0) {
      beforeSeq = this.data.transfers[this.data.transfers.length - 1].sequenceNo;
    }

    const myPlayerId = currentMyPlayerId || this.data.myPlayerId;

    try {
      const list = await getScoreTransfers(this.data.id, beforeSeq, 20);
      const mapped = list.map(t => {
        const timeText = formatTimeOnly(t.createdAt);
        const canReverse =
          this.data.game.status === 'active' &&
          !t.reversedAt &&
          t.transferKind !== 'reversal' &&
          t.receiverPlayerIds.includes(myPlayerId);

        return {
          id: t.id,
          iconCode: t.receiverPlayerIds.length > 1 ? '\uf0c0' : '\uf362',
          text: t.text,
          parsedParts: parseTransferText(t.text),
          time: timeText,
          sequenceNo: t.sequenceNo,
          receiverPlayerIds: t.receiverPlayerIds,
          transferKind: t.transferKind,
          reversalOfTransferId: t.reversalOfTransferId,
          reversedAt: t.reversedAt,
          canReverse: canReverse,
        };
      });
      // ...
  ```

- [ ] **Step 3: Implement onReverseTransfer action handler**
  Add the import of `reverseScoreTransfer` at the top of `apps/miniprogram/src/pages/game-detail/index.ts`:
  ```typescript
  import { reverseScoreTransfer } from '../../services/score.service';
  ```
  And add the event handler method inside the `Page` object in `apps/miniprogram/src/pages/game-detail/index.ts` (e.g., after `inputScore()`):
  ```typescript
  async onReverseTransfer(e: any) {
    const transferId = e.currentTarget.dataset.transferId;
    if (!transferId) return;

    const self = this;
    wx.showModal({
      title: '确认撤销计分',
      content: '确定要撤销这笔计分吗？撤销后，各接收者的得分将被扣回，并返还发送者的得分。',
      confirmText: '确认撤销',
      confirmColor: '#ba1a1a',
      cancelText: '取消',
      success: async (res) => {
        if (res.confirm) {
          wx.showLoading({ title: '撤销中...' });
          try {
            const idempotencyKey = `reverse_${transferId}_${Date.now()}`;
            await reverseScoreTransfer(self.data.id, transferId, idempotencyKey, 'user_reversal');
            wx.showToast({ title: '已撤销', icon: 'success' });
            await self.loadGameData();
          } catch (err) {
            console.error('Reverse transfer failed:', err);
            wx.showToast({ title: (err as any).message || '撤销失败', icon: 'none' });
          } finally {
            wx.hideLoading();
          }
        }
      }
    });
  },
  ```

- [ ] **Step 4: Update game-detail/index.wxml to show the undo icon**
  In `apps/miniprogram/src/pages/game-detail/index.wxml`, inside the `transfer-list` block:
  ```wxml
  <!-- Modify: game-detail/index.wxml -->
  <view class="transfer-list">
    <view wx:for="{{transfers}}" wx:key="id" class="transfer-card">
      <text class="iconfont transfer-icon">{{item.iconCode}}</text>
      <view class="transfer-main">
        <view class="transfer-text">
          <block wx:for="{{item.parsedParts}}" wx:key="index" wx:for-item="part">
            <text wx:if="{{part.type === 'name'}}" class="transfer-name-bold">{{part.text}}</text>
            <text wx:elif="{{part.type === 'value'}}" class="transfer-value-highlight">{{part.text}}</text>
            <text wx:else>{{part.text}}</text>
          </block>
        </view>
        <view class="transfer-meta" style="display: flex; align-items: center; justify-content: space-between; margin-top: 8rpx;">
          <view class="transfer-time">{{item.time}}</view>
          <view wx:if="{{game.status === 'active' && !item.reversedAt && item.transferKind !== 'reversal' && item.canReverse}}" 
                class="reverse-action-btn" 
                style="padding: 10rpx 20rpx; margin-right: -10rpx;"
                bindtap="onReverseTransfer" 
                data-transfer-id="{{item.id}}">
            <text class="iconfont" style="color: var(--text-muted); font-size: 28rpx; opacity: 0.6;">{{icons.undo}}</text>
          </view>
        </view>
      </view>
    </view>
  ```

- [ ] **Step 5: Run compilation check**
  Run: `cd /Users/xuanye/workspaces/1round/apps/miniprogram && pnpm run check && pnpm run build`
  Expected: Mini program compiles and builds with zero errors.
