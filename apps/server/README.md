# OneRound Server

## Run

```bash
go mod tidy
go test ./...
go run ./cmd/oneround-server
```

## Config

Copy `config.example.yaml` to `config.yaml` for local overrides. Environment variables with `ONEROUND_` prefix override key runtime settings.

## Migrations

Migrations live in `migrations/` and run automatically on server startup.
