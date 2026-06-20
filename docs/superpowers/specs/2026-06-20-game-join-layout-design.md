# Game Join Layout Design

Fix the layout of the "Join Game" (加入牌局) screen. Currently, the container card displays its children horizontally due to an unintended `display: flex` rule without `flex-direction: column`.

## Proposed Changes

### WeChat Mini Program Styles

#### [MODIFY] [index.wxss](file:///Users/xuanye/workspaces/1round/apps/miniprogram/src/pages/game-join/index.wxss)

- Remove `.join-card` from the horizontal `display: flex` rule at the top of the file (lines 30-40).
- This allows the card elements to stack vertically and respect their configured margins (`margin-top`).

## Verification Plan

### Manual Verification
- Compile and run the Mini Program.
- Navigate to the "Join Game" screen (e.g. by entering a game invitation or code).
- Confirm that:
  - The card elements stack vertically.
  - The text labels (牌局名称, 创建者, 当前参与者, 我的显示名称) are clearly readable and not squished.
  - The layout looks visually balanced and works correctly in different viewport heights.
