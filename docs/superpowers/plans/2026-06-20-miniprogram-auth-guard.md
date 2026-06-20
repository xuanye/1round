# Mini Program Auth Guard Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ensure every Mini Program page requires an authenticated OneRound user except the public settlement share view.

**Architecture:** Centralize login enforcement in `services/auth.service.ts` with `requireLogin()`, then make every protected page call it before reading or mutating protected data. Keep the existing public settlement path in `game-detail` unauthenticated when opened with `shareToken`.

**Tech Stack:** WeChat Mini Program native TypeScript, existing `wx.login()`, existing `services/http.ts`, Node `assert` scripts for lightweight build-time contract tests.

---

## File Structure

- Modify `apps/miniprogram/src/services/auth.service.ts`
  - Add `requireLogin()` returning the stored user after `ensureLogin()`.
  - Rehydrate login if a token exists but user storage is missing.
- Modify `apps/miniprogram/src/pages/home/index.ts`
  - Replace `ensureLogin()` + `getUser()` with `requireLogin()`.
- Modify `apps/miniprogram/src/pages/game-create/index.ts`
  - Add login guard on page load and before submit.
- Modify `apps/miniprogram/src/pages/game-join/index.ts`
  - Replace `ensureLogin()` with `requireLogin()`.
- Modify `apps/miniprogram/src/pages/game-detail/index.ts`
  - Replace protected path `ensureLogin()` with `requireLogin()`.
  - Preserve `shareToken` public path without login.
- Modify `apps/miniprogram/src/pages/player-manage/index.ts`
  - Add login guard on load and before submit.
- Modify `apps/miniprogram/src/pages/score-input/index.ts`
  - Replace `ensureLogin()` with `requireLogin()`.
- Modify `apps/miniprogram/src/pages/ranking/index.ts`
  - Replace `ensureLogin()` with `requireLogin()`.
- Modify `apps/miniprogram/src/pages/history/index.ts`
  - Replace `ensureLogin()` with `requireLogin()`.
- Create `apps/miniprogram/scripts/auth-guard.test.js`
  - Static contract test: all protected pages import/call `requireLogin()`.
  - Static contract test: `game-detail` keeps `shareToken` public branch before login.
- Modify `apps/miniprogram/package.json`
  - Run auth guard test from `npm test`.
- Modify `apps/miniprogram/README.md`
  - Document protected-page rule and public share exception.

## Auth Contract

Protected pages:

```text
pages/home/index
pages/game-create/index
pages/game-join/index
pages/game-detail/index without shareToken
pages/player-manage/index
pages/score-input/index
pages/ranking/index
pages/history/index
```

Public page mode:

```text
pages/game-detail/index?shareToken=<publicShareToken>
```

Auth identity means the app has:

```ts
{
  token: string;
  user: {
    id: string;
    displayName: string | null;
    avatarUrl: string | null;
  };
}
```

This does not mean WeChat nickname/avatar authorization. It is the OneRound user identity created from `wx.login()` and backend `/api/auth/wechat-login`.

---

### Task 1: Add Static Auth Guard Test

**Files:**
- Create: `apps/miniprogram/scripts/auth-guard.test.js`
- Modify: `apps/miniprogram/package.json`

- [ ] **Step 1: Write the failing auth guard test**

Create `apps/miniprogram/scripts/auth-guard.test.js`:

```js
const assert = require("assert");
const fs = require("fs");
const path = require("path");

const projectRoot = path.resolve(__dirname, "..");
const srcRoot = path.join(projectRoot, "src");

const protectedPages = [
  "pages/home/index.ts",
  "pages/game-create/index.ts",
  "pages/game-join/index.ts",
  "pages/player-manage/index.ts",
  "pages/score-input/index.ts",
  "pages/ranking/index.ts",
  "pages/history/index.ts",
];

for (const page of protectedPages) {
  const filePath = path.join(srcRoot, page);
  const source = fs.readFileSync(filePath, "utf8");
  assert.match(source, /requireLogin/, `${page} must call requireLogin()`);
  assert.match(
    source,
    /from ['"]\.\.\/\.\.\/services\/auth\.service['"]/,
    `${page} must import requireLogin from auth.service`,
  );
}

const gameDetail = fs.readFileSync(path.join(srcRoot, "pages/game-detail/index.ts"), "utf8");
assert.match(gameDetail, /requireLogin/, "game-detail protected mode must call requireLogin()");
assert.match(gameDetail, /this\.data\.shareToken/, "game-detail must preserve shareToken public mode");
assert(
  gameDetail.indexOf("this.data.shareToken") < gameDetail.indexOf("requireLogin"),
  "game-detail must check shareToken before requireLogin()",
);
assert.match(gameDetail, /getPublicSettlement/, "game-detail must keep public settlement loading");
```

- [ ] **Step 2: Wire the test into `npm test`**

Modify `apps/miniprogram/package.json`:

```json
{
  "name": "oneround-miniprogram",
  "version": "0.1.0",
  "private": true,
  "scripts": {
    "build": "node scripts/build.js",
    "check": "tsc -p tsconfig.json --noEmit",
    "test": "node scripts/build.test.js && node scripts/auth-guard.test.js",
    "watch": "node scripts/build.js --watch"
  },
  "devDependencies": {
    "miniprogram-api-typings": "^4.0.7",
    "typescript": "^5.9.3"
  }
}
```

- [ ] **Step 3: Run test to verify it fails**

Run:

```bash
cd apps/miniprogram
npm test
```

Expected:

```text
AssertionError: pages/game-create/index.ts must call requireLogin()
```

Do not edit page code before seeing this failure.

---

### Task 2: Add `requireLogin()` Service

**Files:**
- Modify: `apps/miniprogram/src/services/auth.service.ts`
- Test: `apps/miniprogram/scripts/auth-guard.test.js`

- [ ] **Step 1: Modify imports**

Replace the existing storage import in `apps/miniprogram/src/services/auth.service.ts`:

```ts
import { request } from './http';
import { setToken, getToken, setUser, getUser } from '../utils/storage';
```

- [ ] **Step 2: Add `requireLogin()`**

Append after `ensureLogin()`:

```ts
export async function requireLogin(): Promise<User> {
  await ensureLogin();

  const user = getUser();
  if (user) {
    return {
      id: user.id,
      displayName: user.displayName,
      avatarUrl: user.avatarUrl,
    };
  }

  await login();
  const refreshedUser = getUser();
  if (!refreshedUser) {
    throw new Error('login required');
  }

  return {
    id: refreshedUser.id,
    displayName: refreshedUser.displayName,
    avatarUrl: refreshedUser.avatarUrl,
  };
}
```

- [ ] **Step 3: Run type check**

Run:

```bash
cd apps/miniprogram
npm run check
```

Expected:

```text
> tsc -p tsconfig.json --noEmit
```

Exit code must be `0`.

---

### Task 3: Guard Existing Authenticated Pages

**Files:**
- Modify: `apps/miniprogram/src/pages/home/index.ts`
- Modify: `apps/miniprogram/src/pages/game-join/index.ts`
- Modify: `apps/miniprogram/src/pages/game-detail/index.ts`
- Modify: `apps/miniprogram/src/pages/score-input/index.ts`
- Modify: `apps/miniprogram/src/pages/ranking/index.ts`
- Modify: `apps/miniprogram/src/pages/history/index.ts`
- Test: `apps/miniprogram/scripts/auth-guard.test.js`

- [ ] **Step 1: Update home imports**

In `apps/miniprogram/src/pages/home/index.ts`, replace:

```ts
import { ensureLogin } from '../../services/auth.service';
import { getCurrentGame, getSummary, getHistory, getHistoryStats, leaveGame } from '../../services/game.service';
import { getUser } from '../../utils/storage';
```

with:

```ts
import { requireLogin } from '../../services/auth.service';
import { getCurrentGame, getSummary, getHistory, getHistoryStats, leaveGame } from '../../services/game.service';
```

- [ ] **Step 2: Update home login flow**

Replace:

```ts
await ensureLogin();
const user = getUser();
this.setData({ userName: user?.displayName || '老书记' });
```

with:

```ts
const user = await requireLogin();
this.setData({ userName: user.displayName || '老书记' });
```

- [ ] **Step 3: Update game join import**

In `apps/miniprogram/src/pages/game-join/index.ts`, replace:

```ts
import { ensureLogin } from '../../services/auth.service';
```

with:

```ts
import { requireLogin } from '../../services/auth.service';
```

Then replace:

```ts
await ensureLogin();
```

with:

```ts
await requireLogin();
```

- [ ] **Step 4: Update game detail import**

In `apps/miniprogram/src/pages/game-detail/index.ts`, replace:

```ts
import { ensureLogin } from '../../services/auth.service';
```

with:

```ts
import { requireLogin } from '../../services/auth.service';
```

Then replace:

```ts
await ensureLogin();
await this.loadGameData();
```

with:

```ts
await requireLogin();
await this.loadGameData();
```

Keep this branch above the login call:

```ts
if (this.data.shareToken) {
  await this.loadPublicSettlement();
  return;
}
```

- [ ] **Step 5: Update score input import and call**

In `apps/miniprogram/src/pages/score-input/index.ts`, replace:

```ts
import { ensureLogin } from '../../services/auth.service';
```

with:

```ts
import { requireLogin } from '../../services/auth.service';
```

Then replace:

```ts
await ensureLogin();
```

with:

```ts
await requireLogin();
```

- [ ] **Step 6: Update ranking import and call**

In `apps/miniprogram/src/pages/ranking/index.ts`, replace:

```ts
import { ensureLogin } from '../../services/auth.service';
```

with:

```ts
import { requireLogin } from '../../services/auth.service';
```

Then replace:

```ts
await ensureLogin();
```

with:

```ts
await requireLogin();
```

- [ ] **Step 7: Update history import and call**

In `apps/miniprogram/src/pages/history/index.ts`, replace:

```ts
import { ensureLogin } from '../../services/auth.service';
```

with:

```ts
import { requireLogin } from '../../services/auth.service';
```

Then replace:

```ts
await ensureLogin();
```

with:

```ts
await requireLogin();
```

- [ ] **Step 8: Run partial verification**

Run:

```bash
cd apps/miniprogram
npm run check
node scripts/auth-guard.test.js
```

Expected:

```text
AssertionError: pages/game-create/index.ts must call requireLogin()
```

Type check must pass before the expected auth guard assertion.

---

### Task 4: Add Guard to Missing Protected Pages

**Files:**
- Modify: `apps/miniprogram/src/pages/game-create/index.ts`
- Modify: `apps/miniprogram/src/pages/player-manage/index.ts`
- Test: `apps/miniprogram/scripts/auth-guard.test.js`

- [ ] **Step 1: Guard game create page**

In `apps/miniprogram/src/pages/game-create/index.ts`, replace the imports:

```ts
import { createGame } from '../../services/game.service';
import { saveRecentSession } from '../../utils/storage';
```

with:

```ts
import { requireLogin } from '../../services/auth.service';
import { createGame } from '../../services/game.service';
import { saveRecentSession } from '../../utils/storage';
```

Add this method inside `Page({ ... })`, before `onNameInput`:

```ts
async onLoad() {
  try {
    await requireLogin();
  } catch (err) {
    wx.showToast({ title: (err as any).message || '登录失败', icon: 'none' });
    wx.redirectTo({ url: '/pages/home/index' });
  }
},
```

At the start of `submit()`, before reading `name`, add:

```ts
await requireLogin();
```

The beginning of `submit()` should become:

```ts
async submit() {
  await requireLogin();

  const name = String(this.data.name).trim();
  if (!name) return wx.showToast({ title: '请输入牌局名称', icon: 'none' });
```

- [ ] **Step 2: Guard player manage page**

In `apps/miniprogram/src/pages/player-manage/index.ts`, replace:

```ts
import { updateMyProfile } from '../../services/game.service';
```

with:

```ts
import { requireLogin } from '../../services/auth.service';
import { updateMyProfile } from '../../services/game.service';
```

Change `onLoad` from:

```ts
onLoad(query: Record<string, string>) {
  const id = query.id || '';
  const displayName = query.displayName ? decodeURIComponent(query.displayName) : '';
  this.setData({ id, displayName });
},
```

to:

```ts
async onLoad(query: Record<string, string>) {
  try {
    await requireLogin();
  } catch (err) {
    wx.showToast({ title: (err as any).message || '登录失败', icon: 'none' });
    wx.redirectTo({ url: '/pages/home/index' });
    return;
  }

  const id = query.id || '';
  const displayName = query.displayName ? decodeURIComponent(query.displayName) : '';
  this.setData({ id, displayName });
},
```

At the start of `submit()`, before reading `displayName`, add:

```ts
await requireLogin();
```

The beginning of `submit()` should become:

```ts
async submit() {
  await requireLogin();

  const displayName = String(this.data.displayName).trim();
  if (!displayName) return wx.showToast({ title: '请输入玩家名称', icon: 'none' });
```

- [ ] **Step 3: Run auth guard test to verify it passes**

Run:

```bash
cd apps/miniprogram
node scripts/auth-guard.test.js
```

Expected: exit code `0` with no output.

---

### Task 5: Document Auth Rule

**Files:**
- Modify: `apps/miniprogram/README.md`

- [ ] **Step 1: Add auth section**

Append after the existing Environment section in `apps/miniprogram/README.md`:

```markdown
## Authentication

All Mini Program pages require a OneRound login identity except the public settlement share view.

Protected pages call `requireLogin()` from `src/services/auth.service.ts` before loading protected game, score, ranking, or history data.

Public exception:

```text
pages/game-detail/index?shareToken=<publicShareToken>
```

That path uses `getPublicSettlement(..., auth: false)` and must not force login.

The login identity is created from `wx.login()` plus the backend `/api/auth/wechat-login` response. Do not treat this as WeChat profile authorization for nickname or avatar.
```

- [ ] **Step 2: Run README sanity check**

Run:

```bash
rg -n "Authentication|requireLogin|shareToken|wechat-login" apps/miniprogram/README.md
```

Expected output includes all four terms.

---

### Task 6: Final Verification

**Files:**
- Verify all files changed in prior tasks.

- [ ] **Step 1: Run Mini Program tests**

Run:

```bash
cd apps/miniprogram
npm test
```

Expected:

```text
> node scripts/build.test.js && node scripts/auth-guard.test.js
```

Exit code must be `0`.

- [ ] **Step 2: Run TypeScript check**

Run:

```bash
cd apps/miniprogram
npm run check
```

Expected:

```text
> tsc -p tsconfig.json --noEmit
```

Exit code must be `0`.

- [ ] **Step 3: Build runtime output**

Run:

```bash
cd apps/miniprogram
npm run build
```

Expected:

```text
> node scripts/build.js
```

Exit code must be `0`.

- [ ] **Step 4: Confirm public share path remains unauthenticated**

Run:

```bash
rg -n "shareToken|requireLogin|getPublicSettlement|auth: false" apps/miniprogram/src/pages/game-detail/index.ts apps/miniprogram/src/services/game.service.ts
```

Expected facts:

- `game-detail/index.ts` checks `this.data.shareToken` before `requireLogin()`.
- `game-detail/index.ts` still calls `getPublicSettlement`.
- `game.service.ts` still calls public settlement with `auth: false`.

- [ ] **Step 5: Check git diff**

Run:

```bash
git diff -- apps/miniprogram/src/services/auth.service.ts apps/miniprogram/src/pages apps/miniprogram/scripts/auth-guard.test.js apps/miniprogram/package.json apps/miniprogram/README.md
```

Expected:

- No backend files changed for this task.
- No generated `apps/miniprogram/dist/` files staged.
- No `.env.local` included.

---

## Self-Review

Spec coverage:

- All protected pages are listed and covered by Tasks 3 and 4.
- Public settlement share exception is covered by Tasks 1, 3, and 6.
- User identity semantics are documented in Task 5.
- Verification includes static auth guard test, TypeScript check, and build.

Placeholder scan:

- No placeholder markers or undefined follow-up steps.
- Every code-changing step includes exact target code.

Type consistency:

- `requireLogin(): Promise<User>` returns the existing `models/user` type.
- Page imports consistently use `../../services/auth.service`.
- Existing public API `getPublicSettlement(..., auth: false)` remains unchanged.

Commit policy:

- This repository says not to commit unless explicitly instructed. This plan intentionally omits commit commands. If the user explicitly asks for commits later, commit after Task 6 passes.
