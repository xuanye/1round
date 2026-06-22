# Zap & Lumberjack Logger Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace default slog logger with a structured Zap + Lumberjack logger configuration and update dependencies/interfaces across all packages.

**Architecture:** Define a clean `Logger` interface in `internal/infra/logger` and implement it using a Zap core. Wrap Lumberjack as a file write syncer. Modify the configuration struct to add log settings, and inject the logger into router, middleware, and services.

**Tech Stack:** Go 1.24, go.uber.org/zap, gopkg.in/natefinch/lumberjack.v2

## Global Constraints

- Implement using standard Go packages and Uber's Zap logging libraries.
- The default log output path is at the same level as the application directory (CWD), which is `./logs/oneround.log`.
- Logger method calls in other packages must be adapted to use Zap's typed fields (e.g. `zap.Error`, `zap.String`) instead of raw key-value arguments.

---

### Task 1: Add Dependencies and Modify Config

**Files:**
- Modify: `apps/server/go.mod`
- Modify: `apps/server/internal/config/config.go`
- Modify: `apps/server/config.example.yaml`
- Modify: `apps/server/config.yaml`

**Interfaces:**
- Produces: `config.Config.Log` struct fields.

- [ ] **Step 1: Get dependencies**

Run: `go get go.uber.org/zap gopkg.in/natefinch/lumberjack.v2` in `/Users/xuanye/workspaces/1round/apps/server`

- [ ] **Step 2: Modify Config struct and defaults**

Modify `apps/server/internal/config/config.go` to add `Log` structure:
```go
	Log struct {
		Level      string `yaml:"level"`
		OutputPath string `yaml:"output_path"`
	} `yaml:"log"`
```
In `Default()` function:
```go
	cfg.Log.Level = "info"
	cfg.Log.OutputPath = "./logs/oneround.log"
```
In `applyEnv()` function:
```go
	setString("ONEROUND_LOG_LEVEL", &cfg.Log.Level)
	setString("ONEROUND_LOG_OUTPUT_PATH", &cfg.Log.OutputPath)
```

- [ ] **Step 3: Modify configuration files**

Add `log` section to `apps/server/config.example.yaml` and `apps/server/config.yaml`:
```yaml
log:
  level: info
  output_path: ./logs/oneround.log
```

- [ ] **Step 4: Run go mod tidy and verify**

Run: `go mod tidy` in `/Users/xuanye/workspaces/1round/apps/server`
Verify that `go.mod` includes `go.uber.org/zap` and `gopkg.in/natefinch/lumberjack.v2`.

- [ ] **Step 5: Commit changes**

Run:
```bash
git add go.mod go.sum internal/config/config.go config.example.yaml config.yaml
git commit -m "feat: add logger configuration and zap/lumberjack dependencies"
```

---

### Task 2: Implement Logger Infrastructure

**Files:**
- Create: `apps/server/internal/infra/logger/logger.go`

**Interfaces:**
- Produces: `logger.Logger` interface and factory methods.

- [ ] **Step 1: Implement logger package**

Write the code for `apps/server/internal/infra/logger/logger.go` as defined:
```go
package logger

import (
	"fmt"
	"os"
	"path"
	"runtime"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"github.com/xuanye/one-round/apps/server/internal/config"
)

type Logger interface {
	Info(message string, fields ...zap.Field)
	Debug(message string, fields ...zap.Field)
	Error(message string, fields ...zap.Field)
	Warn(message string, fields ...zap.Field)
	Fatal(message string, fields ...zap.Field)
	InfoF(message string, a ...any)
	DebugF(message string, a ...any)
	ErrorF(message string, a ...any)
	WarnF(message string, a ...any)
	FatalF(message string, a ...any)
	Sync() error
}

type zapLoggerAdapter struct {
	logger *zap.Logger
}

func (log *zapLoggerAdapter) Info(message string, fields ...zap.Field) {
	log.logger.Info(message, fields...)
}

func (log *zapLoggerAdapter) Debug(message string, fields ...zap.Field) {
	callerFields := getCallerInfoForLog()
	fields = append(fields, callerFields...)
	log.logger.Debug(message, fields...)
}

func (log *zapLoggerAdapter) Error(message string, fields ...zap.Field) {
	callerFields := getCallerInfoForLog()
	fields = append(fields, callerFields...)
	log.logger.Error(message, fields...)
}

func (log *zapLoggerAdapter) Warn(message string, fields ...zap.Field) {
	callerFields := getCallerInfoForLog()
	fields = append(fields, callerFields...)
	log.logger.Warn(message, fields...)
}

func (log *zapLoggerAdapter) Fatal(message string, fields ...zap.Field) {
	callerFields := getCallerInfoForLog()
	fields = append(fields, callerFields...)
	log.logger.Fatal(message, fields...)
}

func (log *zapLoggerAdapter) InfoF(message string, a ...any) {
	log.logger.Info(fmt.Sprintf(message, a...))
}

func (log *zapLoggerAdapter) DebugF(message string, a ...any) {
	log.logger.Debug(fmt.Sprintf(message, a...))
}

func (log *zapLoggerAdapter) ErrorF(message string, a ...any) {
	log.logger.Error(fmt.Sprintf(message, a...))
}

func (log *zapLoggerAdapter) WarnF(message string, a ...any) {
	log.logger.Warn(fmt.Sprintf(message, a...))
}

func (log *zapLoggerAdapter) FatalF(message string, a ...any) {
	log.logger.Fatal(fmt.Sprintf(message, a...))
}

func (log *zapLoggerAdapter) Sync() error {
	return log.logger.Sync()
}

func getCallerInfoForLog() (callerFields []zap.Field) {
	pc, file, line, ok := runtime.Caller(2)
	if !ok {
		return
	}
	funcName := runtime.FuncForPC(pc).Name()
	funcName = path.Base(funcName)
	callerFields = append(callerFields, zap.String("func", funcName), zap.String("file", file), zap.Int("line", line))
	return
}

func getFileLogWriter(logFile string) (writeSyncer zapcore.WriteSyncer) {
	today := time.Now().Format("2006-01-02")
	datedFile := fmt.Sprintf("%s-%s.log",
		strings.TrimSuffix(logFile, ".log"),
		today,
	)
	lumberJackLogger := &lumberjack.Logger{
		Filename:   datedFile,
		MaxSize:    100,
		MaxBackups: 60,
		MaxAge:     60,
		Compress:   false,
		LocalTime:  true,
	}
	return zapcore.AddSync(lumberJackLogger)
}

func parseLogLevel(level string) zapcore.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "fatal":
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

func NewZapLoggerAdapter(config *config.Config) Logger {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	consoleEncoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
	encoder := zapcore.NewJSONEncoder(encoderConfig)
	fileWriteSyncer := getFileLogWriter(config.Log.OutputPath)
	zapLevel := parseLogLevel(config.Log.Level)
	core := zapcore.NewTee(
		zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), zapLevel),
		zapcore.NewCore(encoder, fileWriteSyncer, zapLevel),
	)
	rawLogger := zap.New(core)
	return &zapLoggerAdapter{logger: rawLogger}
}

func NewConsole() Logger {
	consoleEncoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
	core := zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), zapcore.DebugLevel)
	return &zapLoggerAdapter{logger: zap.New(core)}
}

func NewNop() Logger {
	return &zapLoggerAdapter{logger: zap.NewNop()}
}
```

- [ ] **Step 2: Commit new package**

Run:
```bash
git add internal/infra/logger/logger.go
git commit -m "feat: implement zap & lumberjack logger adapter"
```

---

### Task 3: Adapt Settlement Service and Auto-Settlement Scheduler

**Files:**
- Modify: `apps/server/internal/app/settlement/service.go`
- Modify: `apps/server/internal/app/scheduler/auto_settlement.go`

**Interfaces:**
- Consumes: `logger.Logger`
- Produces: Updated signatures for `NewService` and `NewAutoSettlementRunner`.

- [ ] **Step 1: Modify settlement/service.go**

Modify `apps/server/internal/app/settlement/service.go`:
Change logger field and constructor:
```go
import (
	...
	"github.com/xuanye/one-round/apps/server/internal/infra/logger"
)

type Service struct {
	...
	logger logger.Logger
}

func NewService(store *sqlite.Store, q *sqlite.Queries, game *gamesvc.Service, hub realtime.Hub, now func() time.Time, log logger.Logger) *Service {
	return &Service{store: store, q: q, game: game, hub: hub, now: now, logger: log}
}
```
Update log call around line 92 (inside `SettleInactiveGames`):
```go
		if err != nil {
			s.logger.Error("auto settlement failed for game", zap.String("game_id", c.ID), zap.Error(err))
			continue
		}
```

- [ ] **Step 2: Modify auto_settlement.go scheduler**

Modify `apps/server/internal/app/scheduler/auto_settlement.go`:
```go
import (
	...
	"github.com/xuanye/one-round/apps/server/internal/infra/logger"
	"go.uber.org/zap"
)

type AutoSettlementRunner struct {
	...
	logger    logger.Logger
}

func NewAutoSettlementRunner(service *settlementsvc.Service, log logger.Logger, interval, threshold time.Duration) *AutoSettlementRunner {
	return &AutoSettlementRunner{
		...
		logger:    log,
	}
}
```
Update logging calls:
```go
					r.logger.Error("auto settlement failed", zap.Error(err))
```
and:
```go
					r.logger.Info("auto settlement completed", zap.Int("finished", result.Finished), zap.Int("voided", result.Voided))
```

- [ ] **Step 3: Commit changes**

Run:
```bash
git add internal/app/settlement/service.go internal/app/scheduler/auto_settlement.go
git commit -m "feat: adapt settlement service and auto-settlement scheduler to custom logger"
```

---

### Task 4: Adapt API Router and Middlewares

**Files:**
- Modify: `apps/server/internal/api/middleware/recover_middleware.go`
- Modify: `apps/server/internal/api/middleware/request_log_middleware.go`
- Modify: `apps/server/internal/api/router.go`

**Interfaces:**
- Consumes: `logger.Logger`
- Produces: Updated signatures for `Recover`, `RequestLog` and `NewRouter`.

- [ ] **Step 1: Modify recover_middleware.go**

Modify `apps/server/internal/api/middleware/recover_middleware.go`:
```go
import (
	...
	"github.com/xuanye/one-round/apps/server/internal/infra/logger"
	"go.uber.org/zap"
)

func Recover(log logger.Logger) func(http.Handler) http.Handler {
	...
					log.Error("request panic", zap.Any("value", recovered))
```

- [ ] **Step 2: Modify request_log_middleware.go**

Modify `apps/server/internal/api/middleware/request_log_middleware.go`:
```go
import (
	...
	"github.com/xuanye/one-round/apps/server/internal/infra/logger"
	"go.uber.org/zap"
)

func RequestLog(log logger.Logger) func(http.Handler) http.Handler {
	...
			log.Info("request", zap.String("method", r.Method), zap.String("path", r.URL.Path), zap.Duration("duration", time.Since(start)))
```

- [ ] **Step 3: Modify router.go**

Modify `apps/server/internal/api/router.go`:
```go
import (
	...
	"github.com/xuanye/one-round/apps/server/internal/infra/logger"
)

func NewRouter(log logger.Logger, services Services) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recover(log))
	r.Use(middleware.RequestLog(log))
    ...
```

- [ ] **Step 4: Commit changes**

Run:
```bash
git add internal/api/middleware/recover_middleware.go internal/api/middleware/request_log_middleware.go internal/api/router.go
git commit -m "feat: adapt middleware and router to custom logger"
```

---

### Task 5: Adapt Server Entrypoint and Main Runner

**Files:**
- Modify: `apps/server/cmd/oneround-server/main.go`

**Interfaces:**
- Consumes: `logger.NewZapLoggerAdapter`
- Produces: None.

- [ ] **Step 1: Modify main.go**

Modify `apps/server/cmd/oneround-server/main.go`:
- Import `github.com/xuanye/one-round/apps/server/internal/infra/logger`
- Import `go.uber.org/zap`
- Remove `log/slog` import.
- Replace main logic to use the new logger:
```go
	// Initialize logging
	cfg, err := config.Load(*configPath)
	// Create temporary bootstrap logger to report load config issues if they occur before logger instantiation.
	if err != nil {
		bootstrapLogger := logger.NewConsole()
		bootstrapLogger.Error("load config", zap.Error(err))
		os.Exit(1)
	}
	logAdapter := logger.NewZapLoggerAdapter(&cfg)
```
Replace the standard main logs:
```go
	db, err := sqlite.Open(ctx, cfg.Database.Path)
	if err != nil {
		logAdapter.Error("open database", zap.Error(err))
		os.Exit(1)
	}
	defer func() {
		if err := db.Close(); err != nil {
			logAdapter.Error("close database", zap.Error(err))
		}
	}()
	if err := goose.SetDialect("sqlite"); err != nil {
		logAdapter.Error("set migration dialect", zap.Error(err))
		os.Exit(1)
	}
	if err := goose.Up(db, "migrations"); err != nil {
		logAdapter.Error("run migrations", zap.Error(err))
		os.Exit(1)
	}
	if *migrateOnly {
		return
	}
```
Update wechat client and router setup:
```go
	if !cfg.Wechat.UseFakeAuth {
		if strings.TrimSpace(cfg.Wechat.AppID) == "" || strings.TrimSpace(cfg.Wechat.AppSecret) == "" {
			logAdapter.Error("wechat app_id and app_secret are required when fake auth is disabled")
			os.Exit(1)
		}
		wechatClient = wechat.NewHTTPClient(cfg.Wechat.AppID, cfg.Wechat.AppSecret, "", http.DefaultClient)
	}

	gameService := gamesvc.NewService(store, queries, hub, wechatClient, now)
	queryService := querysvc.NewService(queries, gameService)
	authService := authsvc.NewService(queries, wechatClient, tokens, now)
	playerService := playersvc.NewService(store, queries, gameService, hub, now)
	roundCycleService := roundcycle.NewService(queries, now)
	scoreTransferService := scoretransfersvc.NewService(store, queries, gameService, roundCycleService, hub, now)
	settlementService := settlementsvc.NewService(store, queries, gameService, hub, now, logAdapter)
	wsHandler := wshandler.NewWebSocketHandler(gameService, hub, cfg.Realtime.ClientSendQueueSize, time.Duration(cfg.Realtime.WriteTimeoutSeconds)*time.Second)

	runner := scheduler.NewAutoSettlementRunner(settlementService, logAdapter, cfg.AutoCheckInterval(), cfg.InactivityThreshold())
	runner.Start(ctx)

	router := api.NewRouter(logAdapter, api.Services{
		Auth: authService, Game: gameService, Player: playerService,
		ScoreTransfer: scoreTransferService, Settlement: settlementService, Query: queryService, Tokens: tokens, WebSocket: wsHandler,
	})
	logAdapter.Info("starting server", zap.String("addr", cfg.Server.HTTPAddr))
	server := &http.Server{
		Addr:    cfg.Server.HTTPAddr,
		Handler: router,
	}
	if err := runHTTPServer(ctx, logAdapter, server, 10*time.Second, hub.Close); err != nil {
		logAdapter.Error("server stopped", zap.Error(err))
		os.Exit(1)
	}
```
Update `runHTTPServer` signature and calls:
```go
func runHTTPServer(ctx context.Context, logAdapter logger.Logger, server *http.Server, shutdownTimeout time.Duration, shutdownHooks ...func(context.Context) error) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
	}

	logAdapter.Info("shutting down server")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	for _, hook := range shutdownHooks {
		if err := hook(shutdownCtx); err != nil && !errors.Is(err, context.Canceled) {
			logAdapter.Error("shutdown hook failed", zap.Error(err))
		}
	}
	if err := server.Shutdown(shutdownCtx); err != nil {
		return err
	}

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-shutdownCtx.Done():
		return shutdownCtx.Err()
	}
}
```

- [ ] **Step 2: Commit main.go changes**

Run:
```bash
git add cmd/oneround-server/main.go
git commit -m "feat: integrate custom logger in main server entrypoint"
```

---

### Task 6: Adapt Testing Code

**Files:**
- Modify: `apps/server/tests/api_test.go`

**Interfaces:**
- Consumes: `logger.NewConsole()`
- Produces: None.

- [ ] **Step 1: Modify api_test.go**

Modify `apps/server/tests/api_test.go` to import `github.com/xuanye/one-round/apps/server/internal/infra/logger` and replace all occurrences of `slog.Default()` with `logger.NewConsole()`.
Remove `"log/slog"` from imports.
In `TestFakeAuthCreateJoinAddSubmitSummaryRankingAPI`, `TestLeaveGameAPI`, `TestLeaveRequiresZeroScoreAPI`, `TestCurrentPreviewJoinAndProfileAPI`, `TestJoinMiniProgramCodeAPI`, `TestCurrentGameReturnsNullWhenUserHasNoActiveGame`, `TestGameScoringLifecycleE2E` and `TestPublicSettlementAPI`:
Change:
```go
	router := api.NewRouter(slog.Default(), api.Services{
```
To:
```go
	router := api.NewRouter(logger.NewConsole(), api.Services{
```

- [ ] **Step 2: Commit changes**

Run:
```bash
git add tests/api_test.go
git commit -m "test: adapt API tests to custom logger interface"
```

---

### Task 7: Verification

**Files:**
- None.

- [ ] **Step 1: Run all tests**

Run: `go test ./...` in `/Users/xuanye/workspaces/1round/apps/server`
Expected: PASS

- [ ] **Step 2: Run server and verify log file output**

Run: `go run cmd/oneround-server/main.go --config config.yaml`
Wait 3 seconds and send Ctrl+C to terminate it.
Verify that `./logs/` folder has been created containing `oneround-YYYY-MM-DD.log`.
Verify console output formats correctly.
