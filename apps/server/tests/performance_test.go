package tests

import (
	"context"
	"errors"
	"github.com/xuanye/one-round/apps/server/internal/api"
	wshandler "github.com/xuanye/one-round/apps/server/internal/api/handler"
	jwtauth "github.com/xuanye/one-round/apps/server/internal/infra/auth"
	"github.com/xuanye/one-round/apps/server/internal/infra/logger"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	scoretransfersvc "github.com/xuanye/one-round/apps/server/internal/app/scoretransfer"
	"github.com/xuanye/one-round/apps/server/internal/domain"
)

func TestPerformanceScopesRangeAndAggregatesSharedGames(t *testing.T) {
	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	app := newTestAppAt(t, start)
	ctx := context.Background()
	me, friend, stranger := login(t, app, "performance-me"), login(t, app, "performance-friend"), login(t, app, "performance-stranger")
	settled := func(owner, joiner string, amount int, when time.Time) {
		t.Helper()
		app.setNow(when)
		game := createGame(t, app, owner, nil)
		if _, err := app.game.Join(ctx, joiner, game.InviteCode, "牌友"); err != nil {
			t.Fatal(err)
		}
		receiver, err := app.query.MyParticipant(ctx, joiner, game.ID)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := app.scoreTransfer.Submit(ctx, owner, game.ID, scoretransfersvc.SubmitInput{ReceiverPlayerIDs: []string{receiver.ID}, Amount: amount, IdempotencyKey: game.ID}); err != nil {
			t.Fatal(err)
		}
		if _, err := app.settlement.FinishDirect(ctx, owner, game.ID); err != nil {
			t.Fatal(err)
		}
	}
	settled(me, friend, 20, start.Add(-time.Hour)) // Before the range.
	settled(me, friend, 20, start)                 // Inclusive start; negative personal score.
	settled(friend, me, 60, start.Add(time.Hour))
	settled(friend, stranger, 100, start.Add(2*time.Hour)) // Not a shared game.
	end := start.Add(24 * time.Hour)
	settled(me, friend, 40, end) // Exclusive end.
	// Current games must not contribute, even with historical participants.
	app.setNow(end.Add(time.Hour))
	createGame(t, app, me, nil)
	result, err := app.query.Performance(ctx, me, start, end)
	if err != nil {
		t.Fatal(err)
	}
	if result.TotalGames != 2 || result.TotalScore != 40 || result.Wins != 1 || result.MaxScore != 60 {
		t.Fatalf("unexpected stats: %+v", result)
	}
	if len(result.Trend) != 2 || result.Trend[0].Score != -20 || result.Trend[1].Score != 40 {
		t.Fatalf("unexpected trend: %+v", result.Trend)
	}
	if len(result.Players) != 2 || !result.Players[0].IsMe || result.Players[1].TotalScore != -40 {
		t.Fatalf("unexpected peers: %+v", result.Players)
	}
	if len(result.RecentGames) != 2 || result.RecentGames[0].MyFinalScore != 60 || result.RecentGames[0].ParticipantCount != 2 {
		t.Fatalf("unexpected recent: %+v", result.RecentGames)
	}
	empty, err := app.query.Performance(ctx, stranger, start, start.Add(time.Hour))
	if err != nil || empty.TotalGames != 0 || empty.Players == nil || empty.Trend == nil || empty.RecentGames == nil {
		t.Fatalf("unexpected empty: %+v, %v", empty, err)
	}
	negative, err := app.query.Performance(ctx, me, start, start.Add(time.Minute))
	if err != nil || negative.MaxScore != -20 || negative.Wins != 0 {
		t.Fatalf("negative maximum: %+v, %v", negative, err)
	}
	for _, bounds := range [][2]time.Time{{end, start}, {start, start}, {start, start.Add(368 * 24 * time.Hour)}} {
		if _, err := app.query.Performance(ctx, me, bounds[0], bounds[1]); !errors.Is(err, domain.ErrInvalidArgument) {
			t.Fatalf("invalid bounds: %v", err)
		}
	}
	if _, err := app.query.Performance(ctx, "", start, end); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("anonymous access: %v", err)
	}
}

func TestPerformanceCountsPositiveTiesButNotZeroGames(t *testing.T) {
	start := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	app := newTestAppAt(t, start)
	ctx := context.Background()
	me, peer, payer := login(t, app, "tie-me"), login(t, app, "tie-peer"), login(t, app, "tie-payer")
	game := createGame(t, app, payer, nil)
	for i, user := range []string{me, peer} {
		if _, err := app.game.Join(ctx, user, game.InviteCode, []string{"自己", "朋友"}[i]); err != nil {
			t.Fatal(err)
		}
	}
	mePlayer, _ := app.query.MyParticipant(ctx, me, game.ID)
	peerPlayer, _ := app.query.MyParticipant(ctx, peer, game.ID)
	if _, err := app.scoreTransfer.Submit(ctx, payer, game.ID, scoretransfersvc.SubmitInput{ReceiverPlayerIDs: []string{mePlayer.ID, peerPlayer.ID}, Amount: 20, IdempotencyKey: "tie"}); err != nil {
		t.Fatal(err)
	}
	if _, err := app.settlement.FinishDirect(ctx, payer, game.ID); err != nil {
		t.Fatal(err)
	}
	app.setNow(start.Add(time.Hour))
	zero := createGame(t, app, me, nil)
	if _, err := app.settlement.FinishDirect(ctx, me, zero.ID); err != nil {
		t.Fatal(err)
	}
	result, err := app.query.Performance(ctx, me, start, start.Add(24*time.Hour))
	if err != nil || result.Wins != 1 || result.TotalGames != 2 {
		t.Fatalf("positive ties and zeros: %+v, %v", result, err)
	}
}

func TestPerformanceAPIValidatesAuthenticationAndBounds(t *testing.T) {
	app := newTestApp(t)
	router := api.NewRouter(logger.NewConsole(), api.Services{
		Auth: app.auth, Game: app.game, Player: app.player, ScoreTransfer: app.scoreTransfer, Settlement: app.settlement, Query: app.query,
		Tokens: jwtauth.NewJWTService("test-signing-key", 720*time.Hour), WebSocket: wshandler.NewWebSocketHandler(app.game, app.hub, 4, time.Second),
	})
	token := loginHTTP(t, router, "performance-api")
	for _, tc := range []struct {
		url, token string
		status     int
	}{
		{"/api/history/performance?start=2026-04-01T00:00:00Z&end=2026-10-01T00:00:00Z", "", http.StatusUnauthorized},
		{"/api/history/performance", token, http.StatusBadRequest},
		{"/api/history/performance?start=bad&end=2026-10-01T00:00:00Z", token, http.StatusBadRequest},
		{"/api/history/performance?start=2026-10-01T00:00:00Z&end=2026-04-01T00:00:00Z", token, http.StatusBadRequest},
		{"/api/history/performance?start=2026-04-01T00:00:00Z&end=2026-10-01T00:00:00Z", token, http.StatusOK},
	} {
		req := httptest.NewRequest(http.MethodGet, tc.url, nil)
		if tc.token != "" {
			req.Header.Set("Authorization", "Bearer "+tc.token)
		}
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		if response.Code != tc.status {
			t.Fatalf("%s: want %d, got %d: %s", tc.url, tc.status, response.Code, response.Body.String())
		}
	}
}
