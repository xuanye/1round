@AGENTS.md

# Execution Contract

## Priority

`AGENTS.md` is mandatory project policy. Treat it as active execution rules, not background documentation.

## Common Commands

### Backend (`apps/server`)

| Command | Description |
|---------|-------------|
| `cd apps/server && go mod tidy` | Normalize Go module dependencies |
| `cd apps/server && go test ./...` | Run all backend tests |
| `cd apps/server && go run ./cmd/oneround-server` | Start local API server |
| `cd apps/server && go run ./cmd/oneround-server -migrate-only` | Run migrations only |
| `curl http://localhost:8080/health` | Check local server health |

### Mini Program (`apps/miniprogram`)

| Command | Description |
|---------|-------------|
| `cd apps/miniprogram && pnpm install` | Install mini program dev dependencies |
| `cd apps/miniprogram && pnpm run build` | Compile TypeScript to mini program JS |
| `cd apps/miniprogram && pnpm run check` | Type-check without emitting JS |
| `cd apps/miniprogram && pnpm run watch` | Watch TypeScript compilation |

### Deploy

| Command | Description |
|---------|-------------|
| `docker compose -f deploy/docker-compose.yml up --build` | Run deployment stack locally |
| `nginx -t -c deploy/nginx.conf` | Validate Nginx config when paths are adjusted |

## Architecture Overview

This is a **Go + SQLite + WeChat Mini Program monorepo** for lightweight multiplayer scorekeeping.

**Backend** follows a layered structure:

- **`cmd/oneround-server`** — process entrypoint, config loading, startup, migrate-only mode
- **`internal/api`** — HTTP/WebSocket transport, DTOs, middleware, response helpers, router
- **`internal/app`** — auth, game, player, round, and summary use cases
- **`internal/domain`** — game sessions, players, rounds, users, domain errors
- **`internal/infra`** — SQLite repositories, transactions, JWT, clock, WeChat clients
- **`internal/realtime`** — WebSocket hub, room membership, client lifecycle, event broadcast
- **`migrations`** — append-only SQLite schema migrations
- **`tests`** — API and service coverage

**Mini Program** follows a page/service/model split:

- **`pages`** — page lifecycle, view state, user interaction
- **`components`** — reusable UI blocks
- **`services`** — HTTP, auth, score, game, realtime access
- **`models`** — shared TypeScript contracts
- **`utils`** — local storage and formatting helpers

**Data flow**: Mini Program Page -> Client Service -> HTTP API -> App Service -> Repository -> SQLite.

**Realtime flow**: persisted mutation -> realtime hub event -> client receives invalidation -> client reloads HTTP summary.

## Required Workflow

Before modifying files:

1. Identify which `AGENTS.md` rules apply.
2. Read the closest README or docs for the affected area.
3. For behavior, public API, schema, auth, realtime, or deployment changes, state assumptions first.
4. If the request conflicts with `AGENTS.md`, stop and ask.

While modifying files:

1. Keep changes local and surgical.
2. No opportunistic cleanup.
3. No premature abstractions.
4. Do not change naming, layout, API shape, or architecture unless required.
5. Do not add dependencies unless the benefit is concrete and documented.

After modifying files:

1. For backend changes, run `cd apps/server && go test ./...`.
2. For mini program TypeScript changes, run `cd apps/miniprogram && pnpm run check`.
3. If `.ts` changes must update runtime output, run `cd apps/miniprogram && pnpm run build`.
4. For migration changes, run `cd apps/server && go run ./cmd/oneround-server -migrate-only` against a disposable/local DB when feasible.
5. Report files changed, verification performed, test result, and remaining risks.

## Safety Rules

- Never commit `apps/server/config.yaml`, production secrets, JWT signing keys, WeChat AppSecret, or database files.
- Do not weaken auth checks for convenience in tests.
- Do not expose backend-only secrets or fake-auth switches in the mini program.
- Do not edit existing migrations that may already have been applied.
- Do not make WebSocket delivery the only path for state correctness.
- Do not convert the single-instance design into multi-instance behavior without documenting Redis/NATS or equivalent broadcast requirements.

## Agent Skills

### Issue Tracker

No project-specific issue tracker is configured yet. If issue workflow is needed, ask for the canonical tracker before creating issues.

### Domain Docs

Project context is currently split across root `README.md`, app-level READMEs, `docs/requirements/`, and `docs/adr/`.

### Verification Default

Use the smallest command set that covers the changed surface:

- Backend only: `cd apps/server && go test ./...`
- Mini program type changes: `cd apps/miniprogram && pnpm run check`
- Mini program emitted JS required: `cd apps/miniprogram && pnpm run build`
- Cross-cutting API/client changes: run both backend tests and mini program check
