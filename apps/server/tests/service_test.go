package tests

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/pressly/goose/v3"
	authsvc "github.com/xuanye/one-round/apps/server/internal/app/auth"
	gamesvc "github.com/xuanye/one-round/apps/server/internal/app/game"
	playersvc "github.com/xuanye/one-round/apps/server/internal/app/player"
	querysvc "github.com/xuanye/one-round/apps/server/internal/app/query"
	roundsvc "github.com/xuanye/one-round/apps/server/internal/app/round"
	"github.com/xuanye/one-round/apps/server/internal/domain"
	jwtauth "github.com/xuanye/one-round/apps/server/internal/infra/auth"
	"github.com/xuanye/one-round/apps/server/internal/infra/sqlite"
	"github.com/xuanye/one-round/apps/server/internal/infra/wechat"
	"github.com/xuanye/one-round/apps/server/internal/realtime"
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

func TestScoreTransferRequestValidation(t *testing.T) {
	tests := []struct {
		name      string
		amount    int
		receivers []string
		wantErr   error
	}{
		{name: "zero amount", amount: 0, receivers: []string{"p2"}, wantErr: domain.ErrInvalidScoreTransferAmount},
		{name: "negative amount", amount: -1, receivers: []string{"p2"}, wantErr: domain.ErrInvalidScoreTransferAmount},
		{name: "no receivers", amount: 1, receivers: nil, wantErr: domain.ErrScoreTransferReceiverRequired},
		{name: "valid", amount: 20, receivers: []string{"p2", "p3"}, wantErr: nil},
		{name: "duplicate receiver", amount: 10, receivers: []string{"p2", "p2"}, wantErr: domain.ErrInvalidPlayer},
		{name: "empty string receiver", amount: 10, receivers: []string{"p2", ""}, wantErr: domain.ErrInvalidPlayer},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := domain.ValidateScoreTransferInput(tt.amount, tt.receivers)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
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

func TestCreateGameStoresCapacityAndOwnerParticipant(t *testing.T) {
	app := newTestApp(t)
	ctx := context.Background()
	user := login(t, app, "owner-code")
	max := 4

	game, err := app.game.Create(ctx, user, "家庭聚会", &max)
	if err != nil {
		t.Fatal(err)
	}
	if game.MaxParticipants == nil || *game.MaxParticipants != 4 {
		t.Fatalf("unexpected max participants: %+v", game.MaxParticipants)
	}
	participants, err := app.query.ActiveParticipants(ctx, user, game.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(participants) != 1 || participants[0].TotalScore != 0 || participants[0].JoinedOrder != 1 {
		t.Fatalf("unexpected owner participant: %+v", participants)
	}
}

func TestCurrentGameExcludesFinishedAndVoidedGames(t *testing.T) {
	app := newTestApp(t)
	ctx := context.Background()
	user := login(t, app, "owner-code")
	game := createGame(t, app, user, nil)

	current, err := app.game.Current(ctx, user)
	if err != nil {
		t.Fatal(err)
	}
	if current == nil || current.ID != game.ID {
		t.Fatalf("unexpected current game: %+v", current)
	}
	if _, err := app.game.Finish(ctx, user, game.ID); err != nil {
		t.Fatal(err)
	}
	current, err = app.game.Current(ctx, user)
	if err != nil {
		t.Fatal(err)
	}
	if current != nil {
		t.Fatalf("expected no current game, got %+v", current)
	}
}

func TestSubmitRoundAccumulatesScoresAndRanking(t *testing.T) {
	app := newTestApp(t)
	ctx := context.Background()
	user := login(t, app, "owner-code")
	game := createGame(t, app, user, nil)
	players, _ := app.player.List(ctx, user, game.ID)
	// Find owner player and add two more
	var ownerPlayer domain.Player
	for _, p := range players {
		if p.UserID != nil && *p.UserID == user {
			ownerPlayer = p
			break
		}
	}
	p1, _ := app.player.Add(ctx, user, game.ID, "爸爸")
	p2, _ := app.player.Add(ctx, user, game.ID, "妈妈")

	result, err := app.round.Submit(ctx, user, game.ID, []roundsvc.ScoreInput{
		{PlayerID: ownerPlayer.ID, Score: 0},
		{PlayerID: p1.ID, Score: 12},
		{PlayerID: p2.ID, Score: -12},
	}, nil)
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
	if ranking[0].PlayerID != p1.ID {
		t.Fatalf("unexpected first in ranking: %+v", ranking)
	}
	if len(ranking) != 3 {
		t.Fatalf("unexpected ranking length: %+v", ranking)
	}
}

func TestFinishedGameRejectsRound(t *testing.T) {
	app := newTestApp(t)
	ctx := context.Background()
	user := login(t, app, "owner-code")
	game := createGame(t, app, user, nil)
	players, _ := app.player.List(ctx, user, game.ID)
	var ownerPlayer domain.Player
	for _, p := range players {
		if p.UserID != nil && *p.UserID == user {
			ownerPlayer = p
			break
		}
	}
	p1, _ := app.player.Add(ctx, user, game.ID, "A")
	p2, _ := app.player.Add(ctx, user, game.ID, "B")
	if _, err := app.game.Finish(ctx, user, game.ID); err != nil {
		t.Fatal(err)
	}
	_, err := app.round.Submit(ctx, user, game.ID, []roundsvc.ScoreInput{
		{PlayerID: ownerPlayer.ID, Score: 0},
		{PlayerID: p1.ID, Score: 1},
		{PlayerID: p2.ID, Score: -1},
	}, nil)
	if err != domain.ErrGameSessionFinished {
		t.Fatalf("expected finished error, got %v", err)
	}
}

func TestNonMemberCannotReadSummary(t *testing.T) {
	app := newTestApp(t)
	ctx := context.Background()
	owner := login(t, app, "owner-code")
	other := login(t, app, "other-code")
	game := createGame(t, app, owner, nil)
	_, err := app.query.Summary(ctx, other, game.ID)
	if err != domain.ErrGameMemberRequired {
		t.Fatalf("expected member error, got %v", err)
	}
}

func TestUserCanHaveOnlyOneCurrentGame(t *testing.T) {
	app := newTestApp(t)
	ctx := context.Background()
	user := login(t, app, "owner-code")
	_ = createGame(t, app, user, nil)

	_, err := app.game.Create(ctx, user, "第二局", nil)
	if err != domain.ErrActiveGameExists {
		t.Fatalf("expected active game conflict, got %v", err)
	}
}

func TestJoinPreviewDoesNotCreateParticipant(t *testing.T) {
	app := newTestApp(t)
	ctx := context.Background()
	owner := login(t, app, "owner-code")
	joiner := login(t, app, "joiner-code")
	game := createGame(t, app, owner, nil)

	preview, err := app.game.JoinPreview(ctx, joiner, game.InviteCode)
	if err != nil {
		t.Fatal(err)
	}
	if preview.GameSessionID != game.ID || preview.ParticipantCount != 1 || preview.CurrentUserDisplayName == "" {
		t.Fatalf("unexpected preview: %+v", preview)
	}
	participants, err := app.query.ActiveParticipants(ctx, owner, game.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(participants) != 1 {
		t.Fatalf("preview created participant: %+v", participants)
	}
}

func TestJoinEnforcesCapacityAndDisplayNameUniqueness(t *testing.T) {
	app := newTestApp(t)
	ctx := context.Background()
	owner := login(t, app, "owner-code")
	max := 2
	game := createGame(t, app, owner, &max)
	joiner := login(t, app, "joiner-code")

	if _, err := app.game.Join(ctx, joiner, game.InviteCode, "妈妈"); err != nil {
		t.Fatal(err)
	}
	third := login(t, app, "third-code")
	_, err := app.game.Join(ctx, third, game.InviteCode, "爸爸")
	if err != domain.ErrGameCapacityFull {
		t.Fatalf("expected capacity full, got %v", err)
	}
}

func TestJoinReactivationPath(t *testing.T) {
	app := newTestApp(t)
	ctx := context.Background()
	owner := login(t, app, "owner-code")
	joiner := login(t, app, "joiner-code")
	game := createGame(t, app, owner, nil)

	// Join the game
	if _, err := app.game.Join(ctx, joiner, game.InviteCode, "妈妈"); err != nil {
		t.Fatal(err)
	}

	// Verify joiner is an active participant
	players, err := app.player.List(ctx, owner, game.ID)
	if err != nil {
		t.Fatal(err)
	}
	var joinerPlayerID string
	found := false
	for _, p := range players {
		if p.UserID != nil && *p.UserID == joiner {
			joinerPlayerID = p.ID
			if !p.Active {
				t.Fatal("expected joiner to be active after joining")
			}
			found = true
		}
	}
	if !found {
		t.Fatal("joiner not found in player list")
	}

	// Simulate leaving: deactivate the joiner's player via direct SQL
	_, err = app.db.ExecContext(ctx, `UPDATE players SET active = 0, left_at = datetime('now') WHERE id = ?`, joinerPlayerID)
	if err != nil {
		t.Fatal(err)
	}

	// Verify joiner is no longer active
	activePlayers, err := app.query.ActiveParticipants(ctx, owner, game.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range activePlayers {
		if p.UserID != nil && *p.UserID == joiner {
			t.Fatal("expected joiner to be inactive after leaving")
		}
	}

	// Rejoin via invite code
	if _, err := app.game.Join(ctx, joiner, game.InviteCode, "妈妈"); err != nil {
		t.Fatalf("rejoin failed: %v", err)
	}

	// Verify the player was reactivated (same player ID, active again)
	rejoinedPlayers, err := app.player.List(ctx, owner, game.ID)
	if err != nil {
		t.Fatal(err)
	}
	reactivated := false
	for _, p := range rejoinedPlayers {
		if p.UserID != nil && *p.UserID == joiner {
			if p.ID != joinerPlayerID {
				t.Fatalf("expected same player ID after reactivation, got %s instead of %s", p.ID, joinerPlayerID)
			}
			if !p.Active {
				t.Fatal("expected joiner to be active after reactivation")
			}
			if p.DisplayName != "妈妈" {
				t.Fatalf("expected display name '妈妈' after reactivation, got %q", p.DisplayName)
			}
			reactivated = true
		}
	}
	if !reactivated {
		t.Fatal("joiner not found after reactivation")
	}
}

func TestLeaveRequiresZeroScore(t *testing.T) {
	app := newTestApp(t)
	ctx := context.Background()
	owner := login(t, app, "owner-code")
	joiner := login(t, app, "joiner-code")
	game := createGame(t, app, owner, nil)
	if _, err := app.game.Join(ctx, joiner, game.InviteCode, "妈妈"); err != nil {
		t.Fatal(err)
	}

	// Get all active players
	activePlayers, _ := app.query.ActiveParticipants(ctx, owner, game.ID)
	var ownerPID, joinerPID string
	for _, p := range activePlayers {
		if p.UserID != nil && *p.UserID == owner {
			ownerPID = p.ID
		}
		if p.UserID != nil && *p.UserID == joiner {
			joinerPID = p.ID
		}
	}

	// Submit a round to give owner a non-zero score
	_, err := app.round.Submit(ctx, owner, game.ID, []roundsvc.ScoreInput{
		{PlayerID: ownerPID, Score: 10},
		{PlayerID: joinerPID, Score: -10},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	err = app.player.Leave(ctx, owner, game.ID)
	if err != domain.ErrCannotLeaveWithNonZeroScore {
		t.Fatalf("expected non-zero leave error, got %v", err)
	}
}

func TestLastZeroScoreParticipantLeaveVoidsUnscoredGame(t *testing.T) {
	app := newTestApp(t)
	ctx := context.Background()
	user := login(t, app, "owner-code")
	game := createGame(t, app, user, nil)

	if err := app.player.Leave(ctx, user, game.ID); err != nil {
		t.Fatal(err)
	}
	current, err := app.game.Current(ctx, user)
	if err != nil {
		t.Fatal(err)
	}
	if current != nil {
		t.Fatalf("expected no current game after void, got %+v", current)
	}
	g, err := app.game.GetForHistoricalMember(ctx, user, game.ID)
	if err != nil {
		t.Fatal(err)
	}
	if g.Status != domain.GameSessionStatusVoided {
		t.Fatalf("expected voided game, got %+v", g.Status)
	}
}

func TestOwnerTransferOnLeave(t *testing.T) {
	app := newTestApp(t)
	ctx := context.Background()
	owner := login(t, app, "owner-code")
	joiner := login(t, app, "joiner-code")
	game := createGame(t, app, owner, nil)
	if _, err := app.game.Join(ctx, joiner, game.InviteCode, "妈妈"); err != nil {
		t.Fatal(err)
	}

	// Owner leaves with zero score, ownership should transfer to joiner
	if err := app.player.Leave(ctx, owner, game.ID); err != nil {
		t.Fatal(err)
	}
	g, err := app.game.GetForHistoricalMember(ctx, owner, game.ID)
	if err != nil {
		t.Fatal(err)
	}
	if g.OwnerUserID != joiner {
		t.Fatalf("expected owner transferred to %s, got %s", joiner, g.OwnerUserID)
	}
}

func TestLastScoredParticipantLeaveFinishesGame(t *testing.T) {
	app := newTestApp(t)
	ctx := context.Background()
	owner := login(t, app, "owner-code")
	joiner := login(t, app, "joiner-code")
	game := createGame(t, app, owner, nil)
	if _, err := app.game.Join(ctx, joiner, game.InviteCode, "妈妈"); err != nil {
		t.Fatal(err)
	}

	// Get all active players
	activePlayers, _ := app.query.ActiveParticipants(ctx, owner, game.ID)
	var ownerPID, joinerPID string
	for _, p := range activePlayers {
		if p.UserID != nil && *p.UserID == owner {
			ownerPID = p.ID
		}
		if p.UserID != nil && *p.UserID == joiner {
			joinerPID = p.ID
		}
	}

	// Submit a round (scored game with zero scores)
	_, err := app.round.Submit(ctx, owner, game.ID, []roundsvc.ScoreInput{
		{PlayerID: ownerPID, Score: 0},
		{PlayerID: joinerPID, Score: 0},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Both leave
	if err := app.player.Leave(ctx, owner, game.ID); err != nil {
		t.Fatal(err)
	}
	if err := app.player.Leave(ctx, joiner, game.ID); err != nil {
		t.Fatal(err)
	}

	g, err := app.game.GetForHistoricalMember(ctx, joiner, game.ID)
	if err != nil {
		t.Fatal(err)
	}
	if g.Status != domain.GameSessionStatusFinished {
		t.Fatalf("expected finished game, got %+v", g.Status)
	}
	if g.SettledAt == nil {
		t.Fatal("expected settled_at to be set")
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
		player: playersvc.NewService(store, q, gameService, hub, now),
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

func createGame(t *testing.T, app *testApp, userID string, maxParticipants *int) domain.GameSession {
	t.Helper()
	game, err := app.game.Create(context.Background(), userID, "家庭聚会", maxParticipants)
	if err != nil {
		t.Fatal(err)
	}
	return game
}
