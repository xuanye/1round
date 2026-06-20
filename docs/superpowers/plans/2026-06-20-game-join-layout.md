# Game Join Layout Fix Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix the layout of the game join page so that the information card displays its components stacked vertically instead of squeezed horizontally.

**Architecture:** Remove the `.join-card` selector from the horizontal `display: flex` rules list in `index.wxss`, turning the card into a block-level container whose child components flow vertically.

**Tech Stack:** WeChat native Mini Program (WXSS, WXML)

## Global Constraints
- Do not commit unless explicitly instructed.
- Ensure the TypeScript compilation checks pass with `pnpm run check` or `npm run check` inside `apps/miniprogram`.
- Ensure style changes are clean and preserve the rest of the visual elements.

---

### Task 1: Remove `.join-card` from the horizontal display: flex selector list

**Files:**
- Modify: `apps/miniprogram/src/pages/game-join/index.wxss`

**Interfaces:**
- Consumes: None
- Produces: None

- [ ] **Step 1: Modify index.wxss to remove .join-card from the flex rules selector list**

Edit `apps/miniprogram/src/pages/game-join/index.wxss`. Find the rules from lines 30 to 40:
```css
.join-topbar,
.join-title,
.join-card,
.join-actions,
.join-brand,
.game-info,
.creator-row,
.participant-head,
.avatar-stack {
  display: flex;
}
```
And replace it with:
```css
.join-topbar,
.join-title,
.join-actions,
.join-brand,
.game-info,
.creator-row,
.participant-head,
.avatar-stack {
  display: flex;
}
```

- [ ] **Step 2: Run miniprogram check to verify no TypeScript compilation errors**

Run: `npm run check` (or `pnpm run check`) in `apps/miniprogram`.
Expected: Exit code 0, no typescript errors.

- [ ] **Step 3: Run miniprogram tests**

Run: `npm run test` in `apps/miniprogram`.
Expected: PASS
