# Design Spec: Receiver-Only Score Reversal (Revised)

Implement the score transfer reversal feature where only the receivers of a scoring transfer can initiate the reversal.

## Backend Changes

### 1. Authorization Logic
Modify the authorization logic in `apps/server/internal/app/scoretransfer/service.go#Reverse` method inside the database transaction:
- Remove the old checks allowing the game owner (`session.OwnerUserID == userID`) or original sender (`original.FromPlayerID == callerPlayer.ID`) to reverse.
- Restrict authorization strictly to receivers.
- Pseudo-code:
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

### 2. Testing Update
Update the integration test `TestReversalReopensRoundInsteadOfAdvancing` in `apps/server/tests/round_cycle_test.go`:
- Change the caller of the `Reverse` method from the owner (`owner.ID`) to the receiver's user ID (`u2.ID`), representing player `p2`.

---

## Frontend Changes

### 1. Model Update
Update the `ScoreTransfer` type in `apps/miniprogram/src/models/score-transfer.ts` to include:
- `transferKind?: string;`
- `reversalOfTransferId?: string;`
- `reversedAt?: string;`

### 2. Local DetailTransfer Type Update
Update the local view type `DetailTransfer` in `apps/miniprogram/src/pages/game-detail/index.ts` to include:
- `receiverPlayerIds: string[];`
- `transferKind?: string;`
- `reversalOfTransferId?: string;`
- `reversedAt?: string;`
- `canReverse: boolean;`

### 3. Page Icons Mapping
In `apps/miniprogram/src/pages/game-detail/index.ts` data initialization, add the undo icon to `icons`:
- `undo: '\uf0e2'`

### 4. Page Logic Update
In `apps/miniprogram/src/pages/game-detail/index.ts`:
- Store `myPlayerId: string` initialized to `''` in the page `data`.
- In `loadGameData`:
  - Locate the active player object for the current logged-in user in `summary.players` (where `userId === user.id`).
  - Store this player's ID (`me.id`) as `myPlayerId` in the page `data`. Do NOT use `user.id`.
- In `loadTransfers`:
  - Retrieve the current player ID.
  - Map each `ScoreTransfer` item to `DetailTransfer` and calculate `canReverse`:
    ```typescript
    const canReverse =
      this.data.game.status === 'active' &&
      !t.reversedAt &&
      t.transferKind !== 'reversal' &&
      t.receiverPlayerIds.includes(myPlayerId);
    ```
- Implement `onReverseTransfer(e)` event handler:
  - Extract the `transferId` from dataset.
  - Prompt confirmation with `wx.showModal` detailing that this action reverses scores for all players involved.
  - Call `reverseScoreTransfer(this.data.id, transferId, idempotencyKey, reason)`.
  - Upon successful reversal, call `await this.loadGameData()` to trigger a full refresh (updating the summary, participants' current scores, round status banners, and log list).

### 5. UI Entrypoint (Subtle Icon)
In `apps/miniprogram/src/pages/game-detail/index.wxml`:
- In the active game's transfer card layout, render a subtle icon (using the `undo` icon code from `icons`) next to the timestamp.
- Bind `bindtap="onReverseTransfer"` with `data-transfer-id="{{item.id}}"`.
- Ensure it is only rendered when:
  - `game.status === 'active'`
  - `!item.reversedAt`
  - `item.transferKind !== 'reversal'`
  - `item.canReverse`
- Keep it visually subtle matching the clean aesthetic.

---

## Architectural/Security Note
- `canReverse` on the frontend is strictly for UI styling/visibility.
- The ultimate source of truth is the backend's transactional check.
