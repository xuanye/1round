# Zap & Lumberjack Logger Implementation Design

This document details the design for replacing the default Go `slog` logger with a custom, structured logger backed by Uber's Zap and Lumberjack for log rotation.

## Architecture

We will implement a custom `Logger` interface to abstract logging details from the application services and middleware. This interface will be implemented using a Zap-based adapter, with Lumberjack handles daily log rotation.

```
+-------------------------------------------------------+
|                    Application                        |
|   (main, middleware, settlement service, tests, etc.)  |
+-------------------------------------------------------+
                           |
                           v
              +-------------------------+
              |      logger.Logger      | (Interface)
              +-------------------------+
                           |
                           v
            +-----------------------------+
            |      zapLoggerAdapter       | (Implementation)
            +-----------------------------+
               /                       \
              v                         v
       +-------------+            +--------------+
       |   Stdout    |            |  Lumberjack  | (Rotation)
       |  (Console)  |            +--------------+
       +-------------+                   |
                                         v
                                  +--------------+
                                  |   Log File   |
                                  +--------------+
```

## Proposed Changes

### 1. Configuration Package

#### [MODIFY] [config.go](file:///Users/xuanye/workspaces/1round/apps/server/internal/config/config.go)
- Add `Log` struct to `Config` with `Level` and `OutputPath` fields.
- Set defaults in `Default()`:
  - `cfg.Log.Level = "info"`
  - `cfg.Log.OutputPath = "./logs/oneround.log"` (relative to current working directory).
- Bind environment variables in `applyEnv()`:
  - `ONEROUND_LOG_LEVEL` -> `cfg.Log.Level`
  - `ONEROUND_LOG_OUTPUT_PATH` -> `cfg.Log.OutputPath`

#### [MODIFY] [config.example.yaml](file:///Users/xuanye/workspaces/1round/apps/server/config.example.yaml)
- Add the `log` section.

#### [MODIFY] [config.yaml](file:///Users/xuanye/workspaces/1round/apps/server/config.yaml)
- Add the `log` section.

### 2. Logger Infrastructure Package

#### [NEW] [logger.go](file:///Users/xuanye/workspaces/1round/apps/server/internal/infra/logger/logger.go)
Create the package `github.com/xuanye/one-round/apps/server/internal/infra/logger`:
- Define `Logger` interface.
- Implement `zapLoggerAdapter` wrapping a `*zap.Logger`.
- Add helper functions:
  - `NewZapLoggerAdapter(config *config.Config) Logger`
  - `NewConsole() Logger` (console-only logger, useful for tests).
  - `NewNop() Logger` (no-op logger, useful for fallbacks).
  - `getCallerInfoForLog()` (adds file, line, and function fields for Debug, Error, Warn, Fatal logs).
  - `getFileLogWriter(logFile string) zapcore.WriteSyncer` (Lumberjack log rotater).

### 3. Application Components Integration

We will replace the usages of `*slog.Logger` with the custom `logger.Logger` interface.

#### [MODIFY] [main.go](file:///Users/xuanye/workspaces/1round/apps/server/cmd/oneround-server/main.go)
- Initialize the custom logger using `logger.NewZapLoggerAdapter(cfg)`.
- Pass it to the auto-settlement runner and the router.
- Replace key-value logging calls with Zap fields.

#### [MODIFY] [recover_middleware.go](file:///Users/xuanye/workspaces/1round/apps/server/internal/api/middleware/recover_middleware.go)
- Change parameter type from `*slog.Logger` to `logger.Logger`.
- Adapt recovery logs.

#### [MODIFY] [request_log_middleware.go](file:///Users/xuanye/workspaces/1round/apps/server/internal/api/middleware/request_log_middleware.go)
- Change parameter type from `*slog.Logger` to `logger.Logger`.
- Adapt request logs.

#### [MODIFY] [router.go](file:///Users/xuanye/workspaces/1round/apps/server/internal/api/router.go)
- Update `NewRouter` signature to accept `logger.Logger`.

#### [MODIFY] [auto_settlement.go](file:///Users/xuanye/workspaces/1round/apps/server/internal/app/scheduler/auto_settlement.go)
- Update fields and `NewAutoSettlementRunner` signature to accept `logger.Logger`.

#### [MODIFY] [service.go](file:///Users/xuanye/workspaces/1round/apps/server/internal/app/settlement/service.go)
- Update `logger` field type to `logger.Logger`.
- Update `NewService` constructor to accept `logger logger.Logger`.
- Pass logger parameter when instantiated in `main.go`.

#### [MODIFY] [api_test.go](file:///Users/xuanye/workspaces/1round/apps/server/tests/api_test.go)
- Initialize routers in tests using `logger.NewConsole()` instead of `slog.Default()`.

## Verification Plan

### Automated Tests
- Run `go test ./...` in the server root to ensure compile correctness and test validation.

### Manual Verification
- Start the server using `go run cmd/oneround-server/main.go`.
- Verify that log files are created under `./logs/` with format `oneround-YYYY-MM-DD.log`.
- Verify stdout printing format.
- Test endpoint access and ensure request logs and panics are recorded in both outputs.
