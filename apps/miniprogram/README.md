# OneRound Mini Program

## TypeScript

Install dependencies:

```bash
pnpm install
```

Compile once:

```bash
pnpm run build
```

Watch during development:

```bash
pnpm run watch
```

## Environment

The build script reads API base URL config in this order:

1. `.env`
2. `.env.local`
3. Shell environment variables

`.env.local` is ignored by Git and is intended for local overrides.

Default:

```env
ONEROUND_API_BASE_URL=https://1round.xuanye.wang
```

Local backend:

```bash
ONEROUND_API_BASE_URL=http://localhost:8080 pnpm run build
```

Or create `apps/miniprogram/.env.local`:

```env
ONEROUND_API_BASE_URL=http://localhost:8080
```

Open this directory in WeChat DevTools:

```text
apps/miniprogram
```

The `src/` directory contains source files. Build output is written to `dist/`, which is the Mini Program runtime root and is ignored by Git.

WeChat DevTools should open this folder directly. The project config points `miniprogramRoot` at `dist/`, and the pnpm scripts provide deterministic local checks.

## UI theme and system fonts

Read the repository-root `DESIGN.md` before changing UI. Shared tokens are in `src/theme.wxss`.
The homepage and custom tab bar select fonts from `wx.getDeviceInfo()`:

- iOS: Apple system sans-serif and PingFang SC.
- Android: Noto Sans Chinese and Roboto, falling back to the device sans-serif.
- HarmonyOS (`ohos` or HarmonyOS system text): HarmonyOS Sans SC / HarmonyOS Sans.
- Unknown or unavailable device info: a system-font fallback stack.

No font download is required; unavailable named fonts fall back to the system sans-serif.
Transparent homepage and navigation PNG assets are reproducible with `scripts/generate-ui-assets.py` (Pillow is needed only to regenerate assets).

## Authentication

All Mini Program pages require a OneRound login identity except the public settlement share view.

Protected pages call `requireLogin()` from `src/services/auth.service.ts` before loading protected game, score, ranking, or history data.

Public exception:

```text
pages/game-detail/index?shareToken=<publicShareToken>
```

That path uses `getPublicSettlement(..., auth: false)` and must not force login.

The login identity is created from `wx.login()` plus the backend `/api/auth/wechat-login` response. Do not treat this as WeChat profile authorization for nickname or avatar.
