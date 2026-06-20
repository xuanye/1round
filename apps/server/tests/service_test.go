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
	scoretransfersvc "github.com/xuanye/one-round/apps/server/internal/app/scoretransfer"
	settlementsvc "github.com/xuanye/one-round/apps/server/internal/app/settlement"
	"github.com/xuanye/one-round/apps/server/internal/domain"
	jwtauth "github.com/xuanye/one-round/apps/server/internal/infra/auth"
	"github.com/xuanye/one-round/apps/server/internal/infra/sqlite"
	"github.com/xuanye/one-round/apps/server/internal/infra/wechat"
	"github.com/xuanye/one-round/apps/server/internal/realtime"
)

type testApp struct {
	auth          *authsvc.Service
	game          *gamesvc.Service
	player        *playersvc.Service
	scoreTransfer *scoretransfersvc.Service
	settlement    *settlementsvc.Service
	query         *querysvc.Service
	q             *sqlite.Queries
	hub           *realtime.MemoryHub
	db            *sql.DB
	nowRef        *time.Time
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
	if _, err := app.settlement.FinishDirect(ctx, user, game.ID); err != nil {
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

func TestLoginAssignsDefaultGlobalDisplayName(t *testing.T) {
	app := newTestApp(t)

	result, err := app.auth.LoginWithWechatCode(context.Background(), "owner-code")
	if err != nil {
		t.Fatal(err)
	}
	if result.User.DisplayName == nil || *result.User.DisplayName == "" {
		t.Fatalf("expected default display name, got %+v", result.User)
	}
}

func TestUpdateMyProfileSyncsGlobalDisplayName(t *testing.T) {
	app := newTestApp(t)
	ctx := context.Background()
	user := login(t, app, "owner-code")
	game := createGame(t, app, user, nil)

	updated, err := app.player.UpdateMyProfile(ctx, user, game.ID, "新昵称")
	if err != nil {
		t.Fatal(err)
	}
	if updated.DisplayName != "新昵称" {
		t.Fatalf("unexpected player profile: %+v", updated)
	}

	loginResult, err := app.auth.LoginWithWechatCode(ctx, "owner-code")
	if err != nil {
		t.Fatal(err)
	}
	if loginResult.User.DisplayName == nil || *loginResult.User.DisplayName != "新昵称" {
		t.Fatalf("expected global display name to sync, got %+v", loginResult.User)
	}
}

func TestJoinPreviewUsesGlobalDisplayNameWhenNoHistoricalName(t *testing.T) {
	app := newTestApp(t)
	ctx := context.Background()
	owner := login(t, app, "owner-code")
	joiner := login(t, app, "joiner-code")
	game := createGame(t, app, owner, nil)

	if _, err := app.q.UpdateUserDisplayName(ctx, joiner, stringPtr("全局昵称"), *app.nowRef); err != nil {
		t.Fatal(err)
	}

	preview, err := app.game.JoinPreview(ctx, joiner, game.InviteCode)
	if err != nil {
		t.Fatal(err)
	}
	if preview.CurrentUserDisplayName != "全局昵称" {
		t.Fatalf("expected join preview to use global name, got %+v", preview)
	}
}

func TestSubmitScoreTransferAccumulatesScoresAndRanking(t *testing.T) {
	app := newTestApp(t)
	ctx := context.Background()
	user := login(t, app, "owner-code")
	game := createGame(t, app, user, nil)

	u2 := login(t, app, "u2-code")
	u3 := login(t, app, "u3-code")
	if _, err := app.game.Join(ctx, u2, game.InviteCode, "爸爸"); err != nil {
		t.Fatal(err)
	}
	if _, err := app.game.Join(ctx, u3, game.InviteCode, "妈妈"); err != nil {
		t.Fatal(err)
	}

	p1, _ := app.query.MyParticipant(ctx, u2, game.ID)
	p2, _ := app.query.MyParticipant(ctx, u3, game.ID)

	// Owner gives 12 to p1
	result, err := app.scoreTransfer.Submit(ctx, user, game.ID, scoretransfersvc.SubmitInput{
		ReceiverPlayerIDs: []string{p1.ID},
		Amount:            12,
		IdempotencyKey:    "transfer-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.SequenceNo != 1 || result.Version != 2 {
		t.Fatalf("unexpected result: %+v", result)
	}

	// p2 gives 12 to p1
	result2, err := app.scoreTransfer.Submit(ctx, u3, game.ID, scoretransfersvc.SubmitInput{
		ReceiverPlayerIDs: []string{p1.ID},
		Amount:            12,
		IdempotencyKey:    "transfer-2",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result2.SequenceNo != 2 || result2.Version != 3 {
		t.Fatalf("unexpected result2: %+v", result2)
	}

	summary, err := app.query.Summary(ctx, user, game.ID)
	if err != nil {
		t.Fatal(err)
	}
	// Owner: -12 (gave 12 to p1), p1: +24 (received 12+12), p2: -12 (gave 12 to p1)
	scores := map[string]int{}
	for _, p := range summary.Players {
		scores[p.ID] = p.TotalScore
	}
	if scores[p1.ID] != 24 {
		t.Fatalf("unexpected p1 score: %+v", scores)
	}
	if scores[p2.ID] != -12 {
		t.Fatalf("unexpected p2 score: %+v", scores)
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

func TestFinishedGameRejectsScoreTransfer(t *testing.T) {
	app := newTestApp(t)
	ctx := context.Background()
	user := login(t, app, "owner-code")
	game := createGame(t, app, user, nil)

	u2 := login(t, app, "u2-code")
	if _, err := app.game.Join(ctx, u2, game.InviteCode, "A"); err != nil {
		t.Fatal(err)
	}
	p2, _ := app.query.MyParticipant(ctx, u2, game.ID)

	if _, err := app.settlement.FinishDirect(ctx, user, game.ID); err != nil {
		t.Fatal(err)
	}
	_, err := app.scoreTransfer.Submit(ctx, user, game.ID, scoretransfersvc.SubmitInput{
		ReceiverPlayerIDs: []string{p2.ID},
		Amount:            1,
		IdempotencyKey:    "finished-key",
	})
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

func TestJoinMiniProgramCodeRequiresActiveMemberGame(t *testing.T) {
	app := newTestApp(t)
	ctx := context.Background()
	owner := login(t, app, "owner-code")
	game := createGame(t, app, owner, nil)

	image, err := app.game.JoinMiniProgramCode(ctx, owner, game.ID)
	if err != nil {
		t.Fatalf("JoinMiniProgramCode returned error: %v", err)
	}
	if len(image) == 0 {
		t.Fatal("expected mini program code image bytes")
	}

	outsider := login(t, app, "outsider-code")
	if _, err := app.game.JoinMiniProgramCode(ctx, outsider, game.ID); err != domain.ErrGameMemberRequired {
		t.Fatalf("expected member required error, got %v", err)
	}

	if _, err := app.settlement.FinishDirect(ctx, owner, game.ID); err != nil {
		t.Fatalf("FinishDirect returned error: %v", err)
	}
	if _, err := app.game.JoinMiniProgramCode(ctx, owner, game.ID); err != domain.ErrGameSessionFinished {
		t.Fatalf("expected finished game error, got %v", err)
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

	joinerP, _ := app.query.MyParticipant(ctx, joiner, game.ID)

	// Owner gives 10 to joiner, making owner's score non-zero
	_, err := app.scoreTransfer.Submit(ctx, owner, game.ID, scoretransfersvc.SubmitInput{
		ReceiverPlayerIDs: []string{joinerP.ID},
		Amount:            10,
		IdempotencyKey:    "leave-test",
	})
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

func TestScoreTransferDebitsActorAndCreditsReceivers(t *testing.T) {
	app := newTestApp(t)
	ctx := context.Background()
	owner := login(t, app, "owner-code")
	u2 := login(t, app, "u2-code")
	u3 := login(t, app, "u3-code")
	game := createGame(t, app, owner, nil)
	if _, err := app.game.Join(ctx, u2, game.InviteCode, "妈妈"); err != nil {
		t.Fatal(err)
	}
	if _, err := app.game.Join(ctx, u3, game.InviteCode, "孩子"); err != nil {
		t.Fatal(err)
	}
	p1, _ := app.query.MyParticipant(ctx, owner, game.ID)
	p2, _ := app.query.MyParticipant(ctx, u2, game.ID)
	p3, _ := app.query.MyParticipant(ctx, u3, game.ID)

	result, err := app.scoreTransfer.Submit(ctx, owner, game.ID, scoretransfersvc.SubmitInput{
		ReceiverPlayerIDs: []string{p2.ID, p3.ID},
		Amount:            20,
		IdempotencyKey:    "submit-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.SequenceNo != 1 || result.Version != 2 {
		t.Fatalf("unexpected result: %+v", result)
	}
	summary, err := app.query.Summary(ctx, owner, game.ID)
	if err != nil {
		t.Fatal(err)
	}
	scores := map[string]int{}
	for _, p := range summary.Players {
		scores[p.ID] = p.TotalScore
	}
	if scores[p1.ID] != -40 || scores[p2.ID] != 20 || scores[p3.ID] != 20 {
		t.Fatalf("unexpected scores: %+v", scores)
	}
}

func TestScoreTransferIsIdempotentByUserAndKey(t *testing.T) {
	app := newTestApp(t)
	ctx := context.Background()
	owner := login(t, app, "owner-code")
	joiner := login(t, app, "joiner-code")
	game := createGame(t, app, owner, nil)
	if _, err := app.game.Join(ctx, joiner, game.InviteCode, "妈妈"); err != nil {
		t.Fatal(err)
	}
	p2, _ := app.query.MyParticipant(ctx, joiner, game.ID)
	input := scoretransfersvc.SubmitInput{ReceiverPlayerIDs: []string{p2.ID}, Amount: 10, IdempotencyKey: "same-key"}

	first, err := app.scoreTransfer.Submit(ctx, owner, game.ID, input)
	if err != nil {
		t.Fatal(err)
	}
	second, err := app.scoreTransfer.Submit(ctx, owner, game.ID, input)
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID || first.Version != second.Version {
		t.Fatalf("expected same idempotent result: first=%+v second=%+v", first, second)
	}
	summary, _ := app.query.Summary(ctx, owner, game.ID)
	if summary.ScoreTransferCnt != 1 {
		t.Fatalf("expected one transfer, got %+v", summary)
	}
}

func TestScoreTransferRejectsSelfAndInactiveReceiver(t *testing.T) {
	app := newTestApp(t)
	ctx := context.Background()
	owner := login(t, app, "owner-code")
	joiner := login(t, app, "joiner-code")
	game := createGame(t, app, owner, nil)
	if _, err := app.game.Join(ctx, joiner, game.InviteCode, "妈妈"); err != nil {
		t.Fatal(err)
	}
	ownerPlayer, _ := app.query.MyParticipant(ctx, owner, game.ID)
	joinerPlayer, _ := app.query.MyParticipant(ctx, joiner, game.ID)

	_, err := app.scoreTransfer.Submit(ctx, owner, game.ID, scoretransfersvc.SubmitInput{
		ReceiverPlayerIDs: []string{ownerPlayer.ID}, Amount: 10, IdempotencyKey: "self",
	})
	if err != domain.ErrInvalidPlayer {
		t.Fatalf("expected invalid self receiver, got %v", err)
	}
	if err := app.player.Leave(ctx, joiner, game.ID); err != nil {
		t.Fatal(err)
	}
	_, err = app.scoreTransfer.Submit(ctx, owner, game.ID, scoretransfersvc.SubmitInput{
		ReceiverPlayerIDs: []string{joinerPlayer.ID}, Amount: 10, IdempotencyKey: "inactive",
	})
	if err != domain.ErrParticipantInactive {
		t.Fatalf("expected inactive receiver error, got %v", err)
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

	ownerP, _ := app.query.MyParticipant(ctx, owner, game.ID)
	joinerP, _ := app.query.MyParticipant(ctx, joiner, game.ID)

	// Owner gives 1 to joiner, joiner gives 1 back to owner
	// Net result: both have 0 score, but the game is "scored"
	if _, err := app.scoreTransfer.Submit(ctx, owner, game.ID, scoretransfersvc.SubmitInput{
		ReceiverPlayerIDs: []string{joinerP.ID},
		Amount:            1,
		IdempotencyKey:    "scored-transfer-1",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := app.scoreTransfer.Submit(ctx, joiner, game.ID, scoretransfersvc.SubmitInput{
		ReceiverPlayerIDs: []string{ownerP.ID},
		Amount:            1,
		IdempotencyKey:    "scored-transfer-2",
	}); err != nil {
		t.Fatal(err)
	}

	// Verify both have zero score now
	ownerP2, _ := app.query.MyParticipant(ctx, owner, game.ID)
	joinerP2, _ := app.query.MyParticipant(ctx, joiner, game.ID)
	if ownerP2.TotalScore != 0 {
		t.Fatalf("expected owner score 0, got %d", ownerP2.TotalScore)
	}
	if joinerP2.TotalScore != 0 {
		t.Fatalf("expected joiner score 0, got %d", joinerP2.TotalScore)
	}

	// Both leave with zero score
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

func TestOwnerCanFinishDirectlyAndFreezeGame(t *testing.T) {
	app := newTestApp(t)
	ctx := context.Background()
	owner := login(t, app, "owner-code")
	joiner := login(t, app, "joiner-code")
	game := createGame(t, app, owner, nil)
	if _, err := app.game.Join(ctx, joiner, game.InviteCode, "妈妈"); err != nil {
		t.Fatal(err)
	}

	finished, err := app.settlement.FinishDirect(ctx, owner, game.ID)
	if err != nil {
		t.Fatal(err)
	}
	if finished.Status != domain.GameSessionStatusFinished || finished.SettledAt == nil || finished.PublicShareToken == nil {
		t.Fatalf("unexpected finished game: %+v", finished)
	}
	_, err = app.game.Join(ctx, login(t, app, "third-code"), game.InviteCode, "孩子")
	if err != domain.ErrGameSessionFinished {
		t.Fatalf("expected finished join rejection, got %v", err)
	}
}

func TestNonOwnerCreatesFinishRequestAndOwnerCanReject(t *testing.T) {
	app := newTestApp(t)
	ctx := context.Background()
	owner := login(t, app, "owner-code")
	joiner := login(t, app, "joiner-code")
	game := createGame(t, app, owner, nil)
	if _, err := app.game.Join(ctx, joiner, game.InviteCode, "妈妈"); err != nil {
		t.Fatal(err)
	}

	req, err := app.settlement.RequestFinish(ctx, joiner, game.ID)
	if err != nil {
		t.Fatal(err)
	}
	if req.Status != domain.FinishRequestStatusPending {
		t.Fatalf("unexpected request: %+v", req)
	}
	req, err = app.settlement.RejectFinishRequest(ctx, owner, game.ID, req.ID)
	if err != nil {
		t.Fatal(err)
	}
	if req.Status != domain.FinishRequestStatusRejected {
		t.Fatalf("unexpected rejected request: %+v", req)
	}
}

func TestOnlyOnePendingFinishRequest(t *testing.T) {
	app := newTestApp(t)
	ctx := context.Background()
	owner := login(t, app, "owner-code")
	u2 := login(t, app, "u2-code")
	u3 := login(t, app, "u3-code")
	game := createGame(t, app, owner, nil)
	if _, err := app.game.Join(ctx, u2, game.InviteCode, "妈妈"); err != nil {
		t.Fatal(err)
	}
	if _, err := app.game.Join(ctx, u3, game.InviteCode, "孩子"); err != nil {
		t.Fatal(err)
	}
	if _, err := app.settlement.RequestFinish(ctx, u2, game.ID); err != nil {
		t.Fatal(err)
	}
	_, err := app.settlement.RequestFinish(ctx, u3, game.ID)
	if err != domain.ErrFinishRequestPending {
		t.Fatalf("expected pending request conflict, got %v", err)
	}
}

func TestHistoryOnlyListsSettledGamesForParticipant(t *testing.T) {
	app := newTestApp(t)
	ctx := context.Background()
	owner := login(t, app, "owner-code")
	other := login(t, app, "other-code")
	game := createGame(t, app, owner, nil)
	if _, err := app.settlement.FinishDirect(ctx, owner, game.ID); err != nil {
		t.Fatal(err)
	}

	ownerHistory, err := app.query.History(ctx, owner, nil, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(ownerHistory.Items) != 1 || ownerHistory.Items[0].ID != game.ID {
		t.Fatalf("unexpected owner history: %+v", ownerHistory)
	}
	otherHistory, err := app.query.History(ctx, other, nil, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(otherHistory.Items) != 0 {
		t.Fatalf("unexpected other history: %+v", otherHistory)
	}
}

func TestPublicSettlementShareOmitsAvatarsAndTransferDetails(t *testing.T) {
	app := newTestApp(t)
	ctx := context.Background()
	owner := login(t, app, "owner-code")
	game := createGame(t, app, owner, nil)
	finished, err := app.settlement.FinishDirect(ctx, owner, game.ID)
	if err != nil {
		t.Fatal(err)
	}

	share, err := app.query.PublicSettlement(ctx, *finished.PublicShareToken)
	if err != nil {
		t.Fatal(err)
	}
	if share.GameSessionID != game.ID || len(share.Participants) != 1 {
		t.Fatalf("unexpected share: %+v", share)
	}
	if share.Participants[0].AvatarURL != nil {
		t.Fatalf("public share leaked avatar: %+v", share.Participants[0])
	}
	if len(share.ScoreTransfers) != 0 {
		t.Fatalf("public share leaked transfer details: %+v", share.ScoreTransfers)
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
	hub.BroadcastToGame(context.Background(), "game-1", realtime.Event{Type: realtime.EventScoreTransferSubmitted})
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
	initial := time.Date(2026, 6, 19, 0, 0, 0, 0, time.UTC)
	nowRef := &initial
	now := func() time.Time { return *nowRef }
	tokens := jwtauth.NewJWTService("test-signing-key", 720*time.Hour)
	wechatClient := wechat.FakeClient{}
	gameService := gamesvc.NewService(store, q, hub, wechatClient, now)
	settlementService := settlementsvc.NewService(store, q, gameService, hub, now)
	return &testApp{
		auth:          authsvc.NewService(q, wechatClient, tokens, now),
		game:          gameService,
		player:        playersvc.NewService(store, q, gameService, hub, now),
		scoreTransfer: scoretransfersvc.NewService(store, q, gameService, hub, now),
		settlement:    settlementService,
		query:         querysvc.NewService(q, gameService),
		q:             q,
		hub:           hub,
		db:            db,
		nowRef:        nowRef,
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

func (app *testApp) setNow(t time.Time) {
	*app.nowRef = t
}

func newTestAppAt(t *testing.T, initial time.Time) *testApp {
	t.Helper()
	app := newTestApp(t)
	app.setNow(initial)
	return app
}

func stringPtr(value string) *string {
	return &value
}

func TestAutoVoidUnscoredGameAfter24Hours(t *testing.T) {
	app := newTestAppAt(t, time.Date(2026, 6, 19, 0, 0, 0, 0, time.UTC))
	ctx := context.Background()
	user := login(t, app, "owner-code")
	game := createGame(t, app, user, nil)

	app.setNow(time.Date(2026, 6, 20, 0, 1, 0, 0, time.UTC))
	result, err := app.settlement.SettleInactiveGames(ctx, 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if result.Voided != 1 || result.Finished != 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
	g, err := app.game.GetForHistoricalMember(ctx, user, game.ID)
	if err != nil {
		t.Fatal(err)
	}
	if g.Status != domain.GameSessionStatusVoided {
		t.Fatalf("expected voided, got %+v", g.Status)
	}
}

func TestAutoFinishScoredGameAfter24HoursSinceLastScore(t *testing.T) {
	app := newTestAppAt(t, time.Date(2026, 6, 19, 0, 0, 0, 0, time.UTC))
	ctx := context.Background()
	owner := login(t, app, "owner-code")
	joiner := login(t, app, "joiner-code")
	game := createGame(t, app, owner, nil)
	if _, err := app.game.Join(ctx, joiner, game.InviteCode, "妈妈"); err != nil {
		t.Fatal(err)
	}
	p2, _ := app.query.MyParticipant(ctx, joiner, game.ID)
	if _, err := app.scoreTransfer.Submit(ctx, owner, game.ID, scoretransfersvc.SubmitInput{
		ReceiverPlayerIDs: []string{p2.ID}, Amount: 10, IdempotencyKey: "k1",
	}); err != nil {
		t.Fatal(err)
	}

	app.setNow(time.Date(2026, 6, 20, 0, 1, 0, 0, time.UTC))
	result, err := app.settlement.SettleInactiveGames(ctx, 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if result.Finished != 1 || result.Voided != 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
	g, err := app.game.GetForHistoricalMember(ctx, owner, game.ID)
	if err != nil {
		t.Fatal(err)
	}
	if g.Status != domain.GameSessionStatusFinished || g.SettledAt == nil {
		t.Fatalf("expected finished, got %+v", g)
	}
}
