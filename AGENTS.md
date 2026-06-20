# OneRound

Lightweight scorekeeper for family gatherings and casual card games. Go API + SQLite backend with a native WeChat Mini Program client.

## Role

You are an AI coding agent working inside a small production-oriented monorepo. Optimize for correctness, simple deployment, and predictable maintenance.

## Tech Stack

- **Backend**: Go 1.24, chi, database/sql, modernc.org/sqlite, goose, JWT, nhooyr websocket, slog
- **Frontend**: WeChat native Mini Program, TypeScript, WXML, WXSS
- **Database**: SQLite with goose migrations
- **Realtime**: WebSocket notifications; HTTP summary API remains the source of truth
- **Deploy**: Single Go server behind Nginx HTTPS/WSS, persistent SQLite file, systemd or Docker Compose
- **Build/Test**: `go test ./...`, TypeScript compiler for the mini program

## Architecture

```text
1round/
├─ apps/
│  ├─ server/
│  │  ├─ cmd/oneround-server/       # Server entrypoint and migration-only mode
│  │  ├─ internal/
│  │  │  ├─ api/                    # HTTP/WebSocket handlers, DTOs, middleware, router
│  │  │  ├─ app/                    # Use-case services and query orchestration
│  │  │  ├─ domain/                 # Core entities and domain errors
│  │  │  ├─ infra/                  # SQLite, auth, clock, WeChat clients
│  │  │  ├─ realtime/               # Hub, rooms, clients, events
│  │  │  └─ config/                 # Runtime configuration
│  │  ├─ migrations/                # Goose SQL migrations
│  │  └─ tests/                     # API and service tests
│  │
│  └─ miniprogram/
│     ├─ pages/                     # WeChat pages
│     ├─ components/                # Reusable WXML/WXSS components
│     ├─ services/                  # HTTP, auth, game, score, realtime clients
│     ├─ models/                    # TypeScript data contracts
│     └─ utils/                     # Formatting and local storage helpers
│
├─ deploy/                          # Docker, Nginx, systemd deployment assets
└─ docs/                            # Requirements, ADRs, implementation plans
```

## Operating Rules

1. Read relevant README/docs before editing code.
2. Prefer small, reviewable changes.
3. Do not change architecture casually.
4. Do not introduce dependencies without justification.
5. Do not silently change API contracts, DTOs, storage schema, or WebSocket payloads.
6. Do not commit unless explicitly instructed.
7. Add or update tests for changed backend behavior.
8. Run relevant verification commands before reporting completion.

## Non-Negotiable Rules

1. **HTTP summary is authoritative** — WebSocket only notifies clients to refresh; do not depend on realtime events as durable state.
2. **No secrets in the client** — WeChat AppSecret, JWT signing keys, production domains, and private config stay in backend config or environment variables.
3. **SQLite constraints are real product constraints** — keep writes simple and transactional; do not design multi-instance behavior without adding a broadcast/storage plan.
4. **Domain logic stays out of handlers and pages** — handlers translate transport; pages render and call services; use-case rules belong in `internal/app` or domain code.
5. **Migrations are append-only** — never edit an applied migration to change behavior; create a new migration.
6. **Generated mini program JS is runtime output** — source changes belong in `.ts`, `.wxml`, `.wxss`, and `.json`; keep generated `.js` in sync when required by the task.
7. **Surgical changes** — change only what the task requires; no opportunistic refactoring.
8. **No premature abstraction** — simple, explicit code beats speculative framework layers.
9. **Chinese product context** — user-facing mini program copy may be Chinese; code identifiers and comments should be English unless existing local context requires otherwise.
10. **Security-sensitive auth** — validate JWT handling, room membership, and session ownership when touching auth, game, round, player, or WebSocket code.
11. **Branching for complex changes** — all relatively complex feature modifications must be done in a dedicated branch, using the format `feature/{feature-name}`.

## Backend Principles

- Keep `internal/domain` independent from API and infrastructure details.
- Keep use-case orchestration in `internal/app`.
- Keep HTTP DTOs under `internal/api/dto`; do not leak persistence structs as API contracts.
- Access SQLite only through repository/transaction modules under `internal/infra/sqlite`.
- Use `context.Context` through request, service, and repository boundaries.
- Prefer explicit SQL and transactions for multi-table updates.
- Use `slog` for structured server logs; avoid ad hoc `fmt.Println` diagnostics.
- Validate request ownership and game membership before mutating shared game state.

## Mini Program Principles

- Treat `.ts` files as source of truth for client logic.
- Keep API calls inside `services/`; pages should not construct raw request URLs repeatedly.
- Keep reusable data shapes in `models/`.
- Keep reusable presentation in `components/`; avoid copying WXML/WXSS across pages.
- Keep local storage access in `utils/storage`.
- Prefer explicit loading, empty, error, and offline states for pages that call the backend.
- Production builds must use HTTPS/WSS domains configured in WeChat Mini Program admin.

## Data and Realtime Rules

- Round submission must persist first, then notify connected clients.
- WebSocket events should be small invalidation signals, not full state replication.
- Clients should reload summary after `round.submitted` or similar room events.
- Score calculations must be deterministic from persisted rounds.
- Avoid financial-settlement semantics; this product records casual scores only.

## Key Conventions

- Server module path: `github.com/xuanye/one-round/apps/server`
- Server entrypoint: `apps/server/cmd/oneround-server`
- Server config example: `apps/server/config.example.yaml`
- Migrations path: `apps/server/migrations`
- Mini program root opened by WeChat DevTools: `apps/miniprogram`
- Environment variables use the `ONEROUND_` prefix.

## Reporting Format

After each task, report:

- What changed
- Files changed
- Verification commands run
- Test result
- Risks or follow-up items
