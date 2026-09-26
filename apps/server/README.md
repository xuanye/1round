# OneRound Server

## Run

```bash
go mod tidy
go test ./...
go run ./cmd/oneround-server
```

## Config

The server reads `config.yaml` by default. This file is ignored by Git and should hold local or production secrets only on the target machine.

Use `config.example.yaml` as the committed reference. Environment variables with `ONEROUND_` prefix override key runtime settings.

Required for real WeChat login:

```yaml
wechat:
  app_id: "wx..."
  app_secret: "..."
  use_fake_auth: false
```

Never put `app_secret` or production JWT signing keys in Mini Program source files.

## Migrations

Migrations live in `migrations/` and run automatically on server startup.

## Game preset scores

`POST /api/game-sessions` accepts optional `presetScores`: 1–8 distinct integers in the range 1–99999, in display order. Omitted or null values use `[20,30,40,60]`; an empty array is invalid. Create, current-game, and summary responses include `presetScores`. The Mini Program exposes four editable values and creates games with `maxParticipants: null`; the existing capacity API remains compatible.

Migration `00008_game_preset_scores.sql` adds persisted game configuration and backfills existing games with the defaults. Start the updated server (which runs migrations) before releasing the client that saves custom values.

## Performance

`GET /api/history/performance?start=<RFC3339>&end=<RFC3339>` requires authentication and returns personal totals, cumulative trend, peers from shared settled games, and the two most recent games in the period. Bounds are inclusive/exclusive and must span no more than 367 days. See `docs/requirements/game-scoring.md` for win and peer-ranking definitions. Deploy the updated server before releasing the performance client; no schema migration is needed.
