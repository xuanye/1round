package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/xuanye/one-round/apps/server/internal/api"
	wshandler "github.com/xuanye/one-round/apps/server/internal/api/handler"
	authsvc "github.com/xuanye/one-round/apps/server/internal/app/auth"
	gamesvc "github.com/xuanye/one-round/apps/server/internal/app/game"
	playersvc "github.com/xuanye/one-round/apps/server/internal/app/player"
	querysvc "github.com/xuanye/one-round/apps/server/internal/app/query"
	roundsvc "github.com/xuanye/one-round/apps/server/internal/app/round"
	scoretransfersvc "github.com/xuanye/one-round/apps/server/internal/app/scoretransfer"
	settlementsvc "github.com/xuanye/one-round/apps/server/internal/app/settlement"
	"github.com/xuanye/one-round/apps/server/internal/app/scheduler"
	"github.com/xuanye/one-round/apps/server/internal/config"
	jwtauth "github.com/xuanye/one-round/apps/server/internal/infra/auth"
	"github.com/xuanye/one-round/apps/server/internal/infra/clock"
	"github.com/xuanye/one-round/apps/server/internal/infra/sqlite"
	"github.com/xuanye/one-round/apps/server/internal/infra/wechat"
	"github.com/xuanye/one-round/apps/server/internal/realtime"
)

func main() {
	configPath := flag.String("config", "config.example.yaml", "config file path")
	migrateOnly := flag.Bool("migrate-only", false, "run migrations and exit")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Error("load config", "error", err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	db, err := sqlite.Open(ctx, cfg.Database.Path)
	if err != nil {
		logger.Error("open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	if err := goose.SetDialect("sqlite"); err != nil {
		logger.Error("set migration dialect", "error", err)
		os.Exit(1)
	}
	if err := goose.Up(db, "migrations"); err != nil {
		logger.Error("run migrations", "error", err)
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
	wechatClient := wechat.FakeClient{}

	gameService := gamesvc.NewService(store, queries, hub, now)
	queryService := querysvc.NewService(queries, gameService)
	authService := authsvc.NewService(queries, wechatClient, tokens, now)
	playerService := playersvc.NewService(store, queries, gameService, hub, now)
	roundService := roundsvc.NewService(store, queries, gameService, hub, now)
	scoreTransferService := scoretransfersvc.NewService(store, queries, gameService, hub, now)
	settlementService := settlementsvc.NewService(store, queries, gameService, hub, now)
	wsHandler := wshandler.NewWebSocketHandler(gameService, hub, cfg.Realtime.ClientSendQueueSize, time.Duration(cfg.Realtime.WriteTimeoutSeconds)*time.Second)

	runner := scheduler.NewAutoSettlementRunner(settlementService, logger, cfg.AutoCheckInterval(), cfg.InactivityThreshold())
	runner.Start(ctx)

	router := api.NewRouter(logger, api.Services{
		Auth: authService, Game: gameService, Player: playerService, Round: roundService,
		ScoreTransfer: scoreTransferService, Settlement: settlementService, Query: queryService, Tokens: tokens, WebSocket: wsHandler,
	})
	logger.Info("starting server", "addr", cfg.Server.HTTPAddr)
	if err := http.ListenAndServe(cfg.Server.HTTPAddr, router); err != nil {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
