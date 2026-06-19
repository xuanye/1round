package tests

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/pressly/goose/v3"
	authsvc "github.com/redhu/one-round/apps/server/internal/app/auth"
	gamesvc "github.com/redhu/one-round/apps/server/internal/app/game"
	playersvc "github.com/redhu/one-round/apps/server/internal/app/player"
	querysvc "github.com/redhu/one-round/apps/server/internal/app/query"
	roundsvc "github.com/redhu/one-round/apps/server/internal/app/round"
	"github.com/redhu/one-round/apps/server/internal/domain"
	jwtauth "github.com/redhu/one-round/apps/server/internal/infra/auth"
	"github.com/redhu/one-round/apps/server/internal/infra/sqlite"
	"github.com/redhu/one-round/apps/server/internal/infra/wechat"
	"github.com/redhu/one-round/apps/server/internal/realtime"
)

type testApp struct {
	auth   *authsvc.Service
	game   *gamesvc.Service
	player *playersvc.Service
	round  *roundsvc.Service
	query  *querysvc.Service
	hub    *realtime.MemoryHub
	db     *sql.DB
}

func TestInviteCodeFormat(t *testing.T) {
	code, err := gamesvc.GenerateInviteCode()
	if err != nil {
		t.Fatal(err)
	}
	if !gamesvc.ValidInviteCode(code) {
		t.Fatalf("invalid invite code: %q", code)
	}
}

func TestZeroSumValidation(t *testing.T) {
	err := roundsvc.ValidateZeroSum([]roundsvc.ScoreInput{{Score: 8}, {Score: -3}})
	if err != domain.ErrScoreTotalMustBeZero {
		t.Fatalf("expected zero sum error, got %v", err)
	}
	if err := roundsvc.ValidateZeroSum([]roundsvc.ScoreInput{{Score: 8}, {Score: -8}}); err != nil {
		t.Fatalf("expected valid zero sum, got %v", err)
	}
}

func TestSubmitRoundAccumulatesScoresAndRanking(t *testing.T) {
	app := newTestApp(t)
	ctx := context.Background()
	user := login(t, app, "owner-code")
	game := createGame(t, app, user, true)
	p1, _ := app.player.Add(ctx, user, game.ID, "爸爸")
	p2, _ := app.player.Add(ctx, user, game.ID, "妈妈")

	result, err := app.round.Submit(ctx, user, game.ID, []roundsvc.ScoreInput{{PlayerID: p1.ID, Score: 12}, {PlayerID: p2.ID, Score: -12}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.RoundNo != 1 || result.Version != 2 {
		t.Fatalf("unexpected round result: %+v", result)
	}
	summary, err := app.query.Summary(ctx, user, game.ID)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Players[0].DisplayName != "爸爸" || summary.Players[0].TotalScore != 12 {
		t.Fatalf("unexpected leading player: %+v", summary.Players)
	}
	ranking, err := app.query.Ranking(ctx, user, game.ID)
	if err != nil {
		t.Fatal(err)
	}
	if ranking[0].PlayerID != p1.ID || ranking[1].PlayerID != p2.ID {
		t.Fatalf("unexpected ranking: %+v", ranking)
	}
}

func TestFinishedGameRejectsRound(t *testing.T) {
	app := newTestApp(t)
	ctx := context.Background()
	user := login(t, app, "owner-code")
	game := createGame(t, app, user, true)
	p1, _ := app.player.Add(ctx, user, game.ID, "A")
	p2, _ := app.player.Add(ctx, user, game.ID, "B")
	if _, err := app.game.Finish(ctx, user, game.ID); err != nil {
		t.Fatal(err)
	}
	_, err := app.round.Submit(ctx, user, game.ID, []roundsvc.ScoreInput{{PlayerID: p1.ID, Score: 1}, {PlayerID: p2.ID, Score: -1}}, nil)
	if err != domain.ErrGameSessionFinished {
		t.Fatalf("expected finished error, got %v", err)
	}
}

func TestNonMemberCannotReadSummary(t *testing.T) {
	app := newTestApp(t)
	ctx := context.Background()
	owner := login(t, app, "owner-code")
	other := login(t, app, "other-code")
	game := createGame(t, app, owner, true)
	_, err := app.query.Summary(ctx, other, game.ID)
	if err != domain.ErrGameMemberRequired {
		t.Fatalf("expected member error, got %v", err)
	}
}

func TestHubBroadcastOnlyTargetsRoom(t *testing.T) {
	hub := realtime.NewMemoryHub()
	c1 := &realtime.Client{Send: make(chan realtime.Event, 1)}
	c2 := &realtime.Client{Send: make(chan realtime.Event, 1)}
	if err := hub.Register(context.Background(), "game-1", c1); err != nil {
		t.Fatal(err)
	}
	if err := hub.Register(context.Background(), "game-2", c2); err != nil {
		t.Fatal(err)
	}
	hub.BroadcastToGame(context.Background(), "game-1", realtime.Event{Type: realtime.EventRoundSubmitted})
	select {
	case <-c1.Send:
	default:
		t.Fatal("expected room client to receive event")
	}
	select {
	case ev := <-c2.Send:
		t.Fatalf("unexpected event in other room: %+v", ev)
	default:
	}
	hub.Unregister("game-1", c1)
}

func newTestApp(t *testing.T) *testApp {
	t.Helper()
	db, err := sqlite.Open(context.Background(), ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if err := goose.SetDialect("sqlite"); err != nil {
		t.Fatal(err)
	}
	if err := goose.Up(db, "../migrations"); err != nil {
		t.Fatal(err)
	}
	q := sqlite.NewQueries(db)
	store := sqlite.NewStore(db)
	hub := realtime.NewMemoryHub()
	now := func() time.Time { return time.Date(2026, 6, 19, 0, 0, 0, 0, time.UTC) }
	tokens := jwtauth.NewJWTService("test-signing-key", 720*time.Hour)
	gameService := gamesvc.NewService(store, q, hub, now)
	return &testApp{
		auth:   authsvc.NewService(q, wechat.FakeClient{}, tokens, now),
		game:   gameService,
		player: playersvc.NewService(q, gameService, hub, now),
		round:  roundsvc.NewService(store, q, gameService, hub, now),
		query:  querysvc.NewService(q, gameService),
		hub:    hub,
		db:     db,
	}
}

func login(t *testing.T, app *testApp, code string) string {
	t.Helper()
	result, err := app.auth.LoginWithWechatCode(context.Background(), code)
	if err != nil {
		t.Fatal(err)
	}
	return result.User.ID
}

func createGame(t *testing.T, app *testApp, userID string, zeroSum bool) domain.GameSession {
	t.Helper()
	game, err := app.game.Create(context.Background(), userID, "家庭聚会", zeroSum)
	if err != nil {
		t.Fatal(err)
	}
	return game
}
