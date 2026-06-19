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
