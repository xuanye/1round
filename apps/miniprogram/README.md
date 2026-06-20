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

## Authentication

All Mini Program pages require a OneRound login identity except the public settlement share view.

Protected pages call `requireLogin()` from `src/services/auth.service.ts` before loading protected game, score, ranking, or history data.

Public exception:

```text
pages/game-detail/index?shareToken=<publicShareToken>
```

That path uses `getPublicSettlement(..., auth: false)` and must not force login.

The login identity is created from `wx.login()` plus the backend `/api/auth/wechat-login` response. Do not treat this as WeChat profile authorization for nickname or avatar.
