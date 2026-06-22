package main

import (
	"context"
	"errors"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/xuanye/one-round/apps/server/internal/api"
	wshandler "github.com/xuanye/one-round/apps/server/internal/api/handler"
	authsvc "github.com/xuanye/one-round/apps/server/internal/app/auth"
	gamesvc "github.com/xuanye/one-round/apps/server/internal/app/game"
	playersvc "github.com/xuanye/one-round/apps/server/internal/app/player"
	querysvc "github.com/xuanye/one-round/apps/server/internal/app/query"
	"github.com/xuanye/one-round/apps/server/internal/app/scheduler"
	"github.com/xuanye/one-round/apps/server/internal/app/roundcycle"
	scoretransfersvc "github.com/xuanye/one-round/apps/server/internal/app/scoretransfer"
	settlementsvc "github.com/xuanye/one-round/apps/server/internal/app/settlement"
	"github.com/xuanye/one-round/apps/server/internal/config"
	jwtauth "github.com/xuanye/one-round/apps/server/internal/infra/auth"
	"github.com/xuanye/one-round/apps/server/internal/infra/clock"
	"github.com/xuanye/one-round/apps/server/internal/infra/logger"
	"github.com/xuanye/one-round/apps/server/internal/infra/sqlite"
	"github.com/xuanye/one-round/apps/server/internal/infra/wechat"
	"github.com/xuanye/one-round/apps/server/internal/realtime"
	"go.uber.org/zap"
)

func main() {
	configPath := flag.String("config", "config.yaml", "config file path")
	migrateOnly := flag.Bool("migrate-only", false, "run migrations and exit")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		bootstrapLogger := logger.NewConsole()
		bootstrapLogger.Error("load config", zap.Error(err))
		os.Exit(1)
	}
	logAdapter := logger.NewZapLoggerAdapter(&cfg)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
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

	utcClock := clock.UTCClock{}
	now := utcClock.Now
	store := sqlite.NewStore(db)
	queries := sqlite.NewQueries(db)
	hub := realtime.NewMemoryHub()
	tokens := jwtauth.NewJWTService(cfg.Auth.SigningKey, cfg.TokenTTL())
	var wechatClient wechat.Client = wechat.FakeClient{}
	if !cfg.Wechat.UseFakeAuth {
		if strings.TrimSpace(cfg.Wechat.AppID) == "" || strings.TrimSpace(cfg.Wechat.AppSecret) == "" {
			logAdapter.Error("wechat app_id and app_secret are required when fake auth is disabled")
			os.Exit(1)
		}
		wechatClient = wechat.NewHTTPClient(cfg.Wechat.AppID, cfg.Wechat.AppSecret, "", http.DefaultClient, logAdapter)
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
}

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
