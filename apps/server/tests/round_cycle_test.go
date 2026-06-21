package tests

import (
	"context"
	"testing"

	"github.com/xuanye/one-round/apps/server/internal/domain"
	scoretransfersvc "github.com/xuanye/one-round/apps/server/internal/app/scoretransfer"
)

type testUser struct {
	ID string
}

func mustLogin(t *testing.T, app *testApp, code string) testUser {
	t.Helper()
	id := login(t, app, code)
	return testUser{ID: id}
}

func mustCreateGameWithOwner(t *testing.T, app *testApp, userID string, name string, maxParticipants *int) (domain.GameSession, domain.Player) {
	t.Helper()
	game := createGame(t, app, userID, maxParticipants)
	p, err := app.query.MyParticipant(context.Background(), userID, game.ID)
	if err != nil {
		t.Fatal(err)
	}
	return game, p
}

func mustJoinGame(t *testing.T, app *testApp, userID string, inviteCode string, displayName string) domain.Player {
	t.Helper()
	gameSessionID, err := app.game.Join(context.Background(), userID, inviteCode, displayName)
	if err != nil {
		t.Fatal(err)
	}
	p, err := app.query.MyParticipant(context.Background(), userID, gameSessionID)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func mustSubmitTransfer(t *testing.T, app *testApp, userID string, gameID string, receiverIDs []string, amount int, idempotencyKey string) domain.ScoreTransfer {
	t.Helper()
	res, err := app.scoreTransfer.Submit(context.Background(), userID, gameID, scoretransfersvc.SubmitInput{
		ReceiverPlayerIDs: receiverIDs,
		Amount:            amount,
		IdempotencyKey:    idempotencyKey,
	})
	if err != nil {
		t.Fatal(err)
	}
	transfer, err := app.q.GetScoreTransfer(context.Background(), res.ID)
	if err != nil {
		t.Fatal(err)
	}
	return transfer
}

func TestRoundStatusCompletesOnlyWhenAllActivePlayersSatisfied(t *testing.T) {
	ctx := context.Background()
	app := newTestApp(t)
	owner := mustLogin(t, app, "owner-code")
	u2 := mustLogin(t, app, "u2-code")
	u3 := mustLogin(t, app, "u3-code")

	game, _ := mustCreateGameWithOwner(t, app, owner.ID, "家庭局", nil)
	p2 := mustJoinGame(t, app, u2.ID, game.InviteCode, "李四")
	p3 := mustJoinGame(t, app, u3.ID, game.InviteCode, "王五")

	if _, err := app.scoreTransfer.Submit(ctx, owner.ID, game.ID, scoretransfersvc.SubmitInput{
		ReceiverPlayerIDs: []string{p2.ID},
		Amount:            10,
		IdempotencyKey:    "t1",
	}); err != nil {
		t.Fatal(err)
	}

	summary, err := app.query.Summary(ctx, owner.ID, game.ID)
	if err != nil {
		t.Fatal(err)
	}
	if summary.RoundStatus.RoundNo != 1 {
		t.Fatalf("expected round 1, got %d", summary.RoundStatus.RoundNo)
	}
	if summary.RoundStatus.Status != "active" {
		t.Fatalf("expected active round, got %s", summary.RoundStatus.Status)
	}
	if len(summary.RoundStatus.PendingPlayerIDs) != 1 || summary.RoundStatus.PendingPlayerIDs[0] != p3.ID {
		t.Fatalf("expected only p3 pending, got %+v", summary.RoundStatus.PendingPlayerIDs)
	}
}

func TestNewRoundOpensOnlyAfterCurrentRoundCompletes(t *testing.T) {
	ctx := context.Background()
	app := newTestApp(t)
	owner := mustLogin(t, app, "owner-code")
	u2 := mustLogin(t, app, "u2-code")
	u3 := mustLogin(t, app, "u3-code")

	game, _ := mustCreateGameWithOwner(t, app, owner.ID, "家庭局", nil)
	p2 := mustJoinGame(t, app, u2.ID, game.InviteCode, "李四")
	_ = mustJoinGame(t, app, u3.ID, game.InviteCode, "王五")

	mustSubmitTransfer(t, app, owner.ID, game.ID, []string{p2.ID}, 10, "t1")
	mustSubmitTransfer(t, app, u3.ID, game.ID, []string{p2.ID}, 20, "t2")

	summary1, err := app.query.Summary(ctx, owner.ID, game.ID)
	if err != nil {
		t.Fatal(err)
	}
	if summary1.RoundStatus.Status != "complete" || !summary1.RoundStatus.CanStartNextRound {
		t.Fatalf("expected completed round status, got %+v", summary1.RoundStatus)
	}

	mustSubmitTransfer(t, app, owner.ID, game.ID, []string{p2.ID}, 5, "t3")
	summary2, err := app.query.Summary(ctx, owner.ID, game.ID)
	if err != nil {
		t.Fatal(err)
	}
	if summary2.RoundStatus.RoundNo != 2 {
		t.Fatalf("expected round 2, got %d", summary2.RoundStatus.RoundNo)
	}
	if summary2.RoundStatus.Status != "active" {
		t.Fatalf("expected active round 2, got %s", summary2.RoundStatus.Status)
	}
}

func TestReversalReopensRoundInsteadOfAdvancing(t *testing.T) {
	ctx := context.Background()
	app := newTestApp(t)
	owner := mustLogin(t, app, "owner-code")
	u2 := mustLogin(t, app, "u2-code")
	u3 := mustLogin(t, app, "u3-code")

	game, _ := mustCreateGameWithOwner(t, app, owner.ID, "家庭局", nil)
	p2 := mustJoinGame(t, app, u2.ID, game.InviteCode, "李四")
	_ = mustJoinGame(t, app, u3.ID, game.InviteCode, "王五")

	first := mustSubmitTransfer(t, app, owner.ID, game.ID, []string{p2.ID}, 10, "t1")

	// Owner (sender) cannot reverse anymore
	if _, err := app.scoreTransfer.Reverse(ctx, owner.ID, game.ID, first.ID, scoretransfersvc.ReverseInput{
		IdempotencyKey: "r-failed-owner",
		Reason:         "attempt_by_sender",
	}); err != domain.ErrForbidden {
		t.Fatalf("expected ErrForbidden for owner, got %v", err)
	}

	// Non-receiver participant (u3) cannot reverse
	if _, err := app.scoreTransfer.Reverse(ctx, u3.ID, game.ID, first.ID, scoretransfersvc.ReverseInput{
		IdempotencyKey: "r-failed-non-receiver",
		Reason:         "attempt_by_non_receiver",
	}); err != domain.ErrForbidden {
		t.Fatalf("expected ErrForbidden for u3, got %v", err)
	}

	revResult, err := app.scoreTransfer.Reverse(ctx, u2.ID, game.ID, first.ID, scoretransfersvc.ReverseInput{
		IdempotencyKey: "r1",
		Reason:         "entered_wrong_receiver",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Attempting to reverse a reversal transfer must fail with ErrConflict
	if _, err := app.scoreTransfer.Reverse(ctx, u2.ID, game.ID, revResult.ID, scoretransfersvc.ReverseInput{
		IdempotencyKey: "r2-on-reversal",
		Reason:         "attempt_to_reverse_reversal",
	}); err != domain.ErrConflict {
		t.Fatalf("expected ErrConflict when reversing a reversal, got %v", err)
	}

	summary, err := app.query.Summary(ctx, owner.ID, game.ID)
	if err != nil {
		t.Fatal(err)
	}
	if summary.RoundStatus.RoundNo != 1 {
		t.Fatalf("expected reversal to stay in round 1, got %d", summary.RoundStatus.RoundNo)
	}
	if summary.RoundStatus.Status != "active" {
		t.Fatalf("expected reopened active round, got %s", summary.RoundStatus.Status)
	}
}

func TestJoinerAddedMidRoundStartsPendingInCurrentRound(t *testing.T) {
	ctx := context.Background()
	app := newTestApp(t)
	owner := mustLogin(t, app, "owner-code")
	u2 := mustLogin(t, app, "u2-code")
	u3 := mustLogin(t, app, "u3-code")
	u4 := mustLogin(t, app, "u4-code")

	game, _ := mustCreateGameWithOwner(t, app, owner.ID, "家庭局", nil)
	p2 := mustJoinGame(t, app, u2.ID, game.InviteCode, "李四")
	_ = mustJoinGame(t, app, u4.ID, game.InviteCode, "赵六")
	mustSubmitTransfer(t, app, owner.ID, game.ID, []string{p2.ID}, 10, "t1")

	p3 := mustJoinGame(t, app, u3.ID, game.InviteCode, "王五")
	summary, err := app.query.Summary(ctx, owner.ID, game.ID)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, id := range summary.RoundStatus.PendingPlayerIDs {
		if id == p3.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected joiner pending in current round, got %+v", summary.RoundStatus.PendingPlayerIDs)
	}
}

func TestLeavingPlayerBecomesExemptFromCurrentRound(t *testing.T) {
	ctx := context.Background()
	app := newTestApp(t)
	owner := mustLogin(t, app, "owner-code")
	u2 := mustLogin(t, app, "u2-code")
	u3 := mustLogin(t, app, "u3-code")

	game, _ := mustCreateGameWithOwner(t, app, owner.ID, "家庭局", nil)
	p2 := mustJoinGame(t, app, u2.ID, game.InviteCode, "李四")
	p3 := mustJoinGame(t, app, u3.ID, game.InviteCode, "王五")

	mustSubmitTransfer(t, app, owner.ID, game.ID, []string{p2.ID}, 10, "t1")
	if err := app.player.Leave(ctx, u3.ID, game.ID); err != nil {
		t.Fatal(err)
	}

	summary, err := app.query.Summary(ctx, owner.ID, game.ID)
	if err != nil {
		t.Fatal(err)
	}
	if summary.RoundStatus.Status != "complete" {
		t.Fatalf("expected round complete after pending player left, got %+v", summary.RoundStatus)
	}
	_ = p3
}
