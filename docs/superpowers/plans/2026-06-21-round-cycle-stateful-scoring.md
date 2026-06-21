# Stateful Round Cycle Scoring Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace heuristic "current round" inference with persisted round-cycle state so the server can accurately tell who has not yet completed the current round, while supporting explicit score correction without accidentally advancing to the next round.

**Architecture:** Add a first-class `round_cycles` aggregate and a per-player `round_participation_statuses` table owned by the backend write path. Each `score_transfer` is assigned to one round at write time, and explicit reversal records mutate round participation state transactionally. The Mini Program consumes summary DTOs only; it no longer derives round completeness from transfer history.

**Tech Stack:** Go 1.24, chi, database/sql, modernc.org/sqlite, goose SQL migrations, existing service/integration test style in `apps/server/tests`, WeChat Mini Program TypeScript.

---

## Scope Check

- This is one coupled server + Mini Program plan because the persistence model, write transaction, summary DTO, and warning UX must ship together.
- This plan intentionally supersedes the current query-time `CalculateRoundStatus(...)` approach.
- This plan changes API and storage semantics. Release server and Mini Program together.

## Design Decisions

1. **Round state is persisted, not inferred.**
   - Add `round_cycles` and `round_participation_statuses`.
   - Add `round_cycle_id` and `transfer_kind` to `score_transfers`.

2. **Correction is explicit, not guessed from reverse score shape.**
   - Add `POST /api/game-sessions/{id}/score-transfers/{transferId}/reversal`.
   - Keep normal `POST /score-transfers` unchanged for non-correction traffic.
   - Do not infer that `B -> A` automatically reverses `A -> B`.

3. **One active round per active game session.**
   - New games start with round 1 in `active` state.
   - After a transfer/reversal updates participation, if all active players are satisfied, mark the round complete.
   - The next normal transfer lazily opens the next round.

4. **Round participation is per active player only.**
   - Active joiners added mid-game join the current active round as `pending`.
   - Players who leave are removed from active-round completeness checks by marking their status as `exempt`.
   - Historical transfers still remain visible in detail/history pages.

## File Structure

- Create: `apps/server/migrations/00007_round_cycles.sql`
  - New tables and schema columns for explicit round state and reversal links.
- Create: `apps/server/internal/domain/round_cycle.go`
  - Canonical round-cycle and participation domain types.
- Create: `apps/server/internal/app/roundcycle/service.go`
  - Round state orchestration helpers used by score transfer and player flows.
- Create: `apps/server/internal/infra/sqlite/round_cycle_repository.go`
  - SQL helpers for round-cycle reads/writes and participation state mutation.
- Create: `apps/server/tests/round_cycle_test.go`
  - Integration tests covering round completion, reversals, joins, and leaves.
- Modify: `apps/server/internal/domain/score_transfer.go`
  - Add persisted round linkage and reversal metadata.
- Modify: `apps/server/internal/app/scoretransfer/service.go`
  - Assign transfers to rounds, mutate participation state, and expose reversal flow.
- Modify: `apps/server/internal/app/player/service.go`
  - Update active round participation when users join/leave.
- Modify: `apps/server/internal/app/game/service.go`
  - Seed round 1 for newly created games.
- Modify: `apps/server/internal/app/query/summary_service.go`
  - Read persisted round state; delete heuristic `CalculateRoundStatus`.
- Modify: `apps/server/internal/app/query/summary_service_test.go`
  - Replace heuristic tests with round-status projection tests backed by persisted state.
- Modify: `apps/server/internal/api/dto/score_transfer_dto.go`
  - Add reversal request/response payloads if needed.
- Modify: `apps/server/internal/api/handler/score_transfer_handler.go`
  - Route and decode reversal requests.
- Modify: `apps/server/internal/api/handler/game_handler.go`
  - Summary response already flows through query service; update only if DTO glue changes.
- Modify: `apps/server/cmd/oneround-server/main.go`
  - Wire round-cycle service into score transfer, player, and game services.
- Modify: `apps/miniprogram/src/models/game-session.ts`
  - Replace inferred `roundStatus` shape with persisted DTO contract.
- Modify: `apps/miniprogram/src/services/game.service.ts`
  - Add reversal API wrapper if Mini Program exposes it immediately.
- Modify: `apps/miniprogram/src/services/score.service.ts`
  - Add reversal request wrapper for correcting a transfer.
- Modify: `apps/miniprogram/src/pages/game-detail/index.ts`
  - Render persisted round status and route to correction flow if exposed.
- Modify: `apps/miniprogram/src/pages/game-detail/index.wxml`
  - Render warning banner from persisted pending-player list.
- Modify: `apps/miniprogram/src/pages/game-detail/index.wxss`
  - Styling only; preserve existing visual language.
- Modify: `apps/miniprogram/src/pages/score-input/index.ts`
  - Keep normal score submission; optionally accept `mode=reverse&transferId=...`.
- Modify: `docs/requirements/game-scoring.md`
  - Add explicit correction semantics and round-cycle contract.

## API Contract

### Summary response

```json
{
  "roundStatus": {
    "roundNo": 3,
    "status": "active",
    "pendingPlayerIds": ["p2", "p4"],
    "pendingPlayerNames": ["李四", "赵六"],
    "canStartNextRound": false
  }
}
```

### Normal transfer request

```json
{
  "receiverPlayerIds": ["p2"],
  "amount": 20,
  "idempotencyKey": "score_transfer_game1_1712345678_abcd123"
}
```

### Reversal request

```json
{
  "idempotencyKey": "reverse_transfer_game1_1712349999_zxy987",
  "reason": "entered_wrong_receiver"
}
```

### Transfer persistence additions

```text
score_transfers.round_cycle_id            NOT NULL
score_transfers.transfer_kind             normal | reversal
score_transfers.reversal_of_transfer_id   NULLABLE UNIQUE
score_transfers.reversed_at               NULLABLE
```

## Migration / Backfill Strategy

- Existing active games need one active round row.
- Existing score transfers should be assigned to a bootstrap round only if we do not support data-perfect historical reconstruction.
- Because old data has no authoritative round semantics, migrate active historical transfers with:
  - one synthetic round row per active game session
  - all existing transfers assigned to that synthetic round
  - all active players in that game marked `satisfied`
- After deploy, only newly written data has perfect round semantics.
- If preserving exact historical current-round reminders for old active sessions matters, stop here and write a one-off admin reconciliation script. Otherwise, accept this forward-only boundary and document it.

---

### Task 1: Lock the Server Contract with Failing Integration Tests

**Files:**
- Create: `apps/server/tests/round_cycle_test.go`
- Modify: `apps/server/internal/app/query/summary_service_test.go`

- [ ] **Step 1: Write failing integration tests for normal round completion**

Create `apps/server/tests/round_cycle_test.go`:

```go
package tests

import (
	"context"
	"testing"

	scoretransfersvc "github.com/xuanye/one-round/apps/server/internal/app/scoretransfer"
)

func TestRoundStatusCompletesOnlyWhenAllActivePlayersSatisfied(t *testing.T) {
	ctx := context.Background()
	app := newTestApp(t)
	owner := mustLogin(t, app, "owner-code")
	u2 := mustLogin(t, app, "u2-code")
	u3 := mustLogin(t, app, "u3-code")

	game, p1 := mustCreateGameWithOwner(t, app, owner.ID, "家庭局", nil)
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

	_ = p1
	_ = p3
}
```

- [ ] **Step 2: Extend the test with round rollover only after completion**

Append to the same test file:

```go
func TestNewRoundOpensOnlyAfterCurrentRoundCompletes(t *testing.T) {
	ctx := context.Background()
	app := newTestApp(t)
	owner := mustLogin(t, app, "owner-code")
	u2 := mustLogin(t, app, "u2-code")
	u3 := mustLogin(t, app, "u3-code")

	game, _ := mustCreateGameWithOwner(t, app, owner.ID, "家庭局", nil)
	p2 := mustJoinGame(t, app, u2.ID, game.InviteCode, "李四")
	p3 := mustJoinGame(t, app, u3.ID, game.InviteCode, "王五")

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

	_ = p3
}
```

- [ ] **Step 3: Write the failing integration test for explicit reversal**

Append:

```go
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
	if _, err := app.scoreTransfer.Reverse(ctx, owner.ID, game.ID, first.ID, scoretransfersvc.ReverseInput{
		IdempotencyKey: "r1",
		Reason:         "entered_wrong_receiver",
	}); err != nil {
		t.Fatal(err)
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
```

- [ ] **Step 4: Replace the heuristic unit test with persisted-state projection tests**

Rewrite `apps/server/internal/app/query/summary_service_test.go` around DTO mapping instead of `CalculateRoundStatus(...)`:

```go
func TestSummaryRoundStatusProjection(t *testing.T) {
	summary := Summary{
		RoundStatus: RoundStatus{
			RoundNo:            4,
			Status:             "active",
			PendingPlayerIDs:   []string{"p2"},
			PendingPlayerNames: []string{"李四"},
			CanStartNextRound:  false,
		},
	}

	if summary.RoundStatus.RoundNo != 4 {
		t.Fatalf("expected round 4, got %d", summary.RoundStatus.RoundNo)
	}
	if summary.RoundStatus.PendingPlayerNames[0] != "李四" {
		t.Fatalf("unexpected pending names: %+v", summary.RoundStatus.PendingPlayerNames)
	}
}
```

- [ ] **Step 5: Run the focused server tests to verify they fail**

Run:

```bash
go test ./tests -run 'Test(RoundStatusCompletesOnlyWhenAllActivePlayersSatisfied|NewRoundOpensOnlyAfterCurrentRoundCompletes|ReversalReopensRoundInsteadOfAdvancing)' -count=1
go test ./internal/app/query -run TestSummaryRoundStatusProjection -count=1
```

Expected:

- `FAIL` because `RoundStatus` fields, `Reverse(...)`, and persisted round infrastructure do not exist yet.

---

### Task 2: Add Round-Cycle Persistence and Backfill Migration

**Files:**
- Create: `apps/server/migrations/00007_round_cycles.sql`
- Create: `apps/server/internal/domain/round_cycle.go`
- Create: `apps/server/internal/infra/sqlite/round_cycle_repository.go`
- Modify: `apps/server/internal/domain/score_transfer.go`

- [ ] **Step 1: Write the migration**

Create `apps/server/migrations/00007_round_cycles.sql`:

```sql
-- +goose Up
CREATE TABLE round_cycles (
    id TEXT PRIMARY KEY,
    game_session_id TEXT NOT NULL,
    round_no INTEGER NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('active', 'complete')),
    created_at TEXT NOT NULL,
    completed_at TEXT NULL,
    UNIQUE(game_session_id, round_no),
    UNIQUE(game_session_id, status) DEFERRABLE INITIALLY DEFERRED,
    FOREIGN KEY(game_session_id) REFERENCES game_sessions(id)
);

CREATE TABLE round_participation_statuses (
    id TEXT PRIMARY KEY,
    round_cycle_id TEXT NOT NULL,
    player_id TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('pending', 'satisfied', 'exempt')),
    satisfied_by_transfer_id TEXT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(round_cycle_id, player_id),
    FOREIGN KEY(round_cycle_id) REFERENCES round_cycles(id),
    FOREIGN KEY(player_id) REFERENCES players(id),
    FOREIGN KEY(satisfied_by_transfer_id) REFERENCES score_transfers(id)
);

ALTER TABLE score_transfers ADD COLUMN round_cycle_id TEXT NULL;
ALTER TABLE score_transfers ADD COLUMN transfer_kind TEXT NOT NULL DEFAULT 'normal';
ALTER TABLE score_transfers ADD COLUMN reversal_of_transfer_id TEXT NULL;
ALTER TABLE score_transfers ADD COLUMN reversed_at TEXT NULL;

CREATE INDEX idx_round_cycles_game_status ON round_cycles(game_session_id, status);
CREATE INDEX idx_round_participation_round_status ON round_participation_statuses(round_cycle_id, status);
CREATE INDEX idx_score_transfers_round_cycle ON score_transfers(round_cycle_id, sequence_no DESC);
CREATE UNIQUE INDEX idx_score_transfers_reversal_of ON score_transfers(reversal_of_transfer_id) WHERE reversal_of_transfer_id IS NOT NULL;

INSERT INTO round_cycles (id, game_session_id, round_no, status, created_at, completed_at)
SELECT hex(randomblob(16)), id, 1, 'complete', updated_at, updated_at
FROM game_sessions
WHERE status = 'active' AND round_count > 0;

INSERT INTO round_cycles (id, game_session_id, round_no, status, created_at, completed_at)
SELECT hex(randomblob(16)), id, 1, 'active', created_at, NULL
FROM game_sessions
WHERE status = 'active' AND round_count = 0;

UPDATE score_transfers
SET round_cycle_id = (
    SELECT rc.id FROM round_cycles rc
    WHERE rc.game_session_id = score_transfers.game_session_id AND rc.round_no = 1
)
WHERE round_cycle_id IS NULL;

-- +goose Down
DROP INDEX idx_score_transfers_reversal_of;
DROP INDEX idx_score_transfers_round_cycle;
DROP INDEX idx_round_participation_round_status;
DROP INDEX idx_round_cycles_game_status;
DROP TABLE round_participation_statuses;
DROP TABLE round_cycles;
```

- [ ] **Step 2: Add domain types**

Create `apps/server/internal/domain/round_cycle.go`:

```go
package domain

import "time"

type RoundCycleStatus string

const (
	RoundCycleStatusActive   RoundCycleStatus = "active"
	RoundCycleStatusComplete RoundCycleStatus = "complete"
)

type ParticipationStatus string

const (
	ParticipationStatusPending   ParticipationStatus = "pending"
	ParticipationStatusSatisfied ParticipationStatus = "satisfied"
	ParticipationStatusExempt    ParticipationStatus = "exempt"
)

type RoundCycle struct {
	ID            string
	GameSessionID string
	RoundNo       int
	Status        RoundCycleStatus
	CreatedAt     time.Time
	CompletedAt   *time.Time
}

type RoundParticipationStatus struct {
	ID                  string
	RoundCycleID        string
	PlayerID            string
	Status              ParticipationStatus
	SatisfiedByTransfer *string
	UpdatedAt           time.Time
}
```

- [ ] **Step 3: Extend score transfer domain metadata**

Modify `apps/server/internal/domain/score_transfer.go`:

```go
type ScoreTransferKind string

const (
	ScoreTransferKindNormal   ScoreTransferKind = "normal"
	ScoreTransferKindReversal ScoreTransferKind = "reversal"
)

type ScoreTransfer struct {
	ID                   string
	GameSessionID        string
	RoundCycleID         string
	SequenceNo           int
	FromPlayerID         string
	CreatedByUserID      string
	IdempotencyKey       string
	Amount               int
	Kind                 ScoreTransferKind
	ReversalOfTransferID *string
	ReversedAt           *time.Time
	CreatedAt            time.Time
	Receivers            []ScoreTransferReceiver
}
```

- [ ] **Step 4: Add repository primitives**

Create `apps/server/internal/infra/sqlite/round_cycle_repository.go` with signatures first:

```go
package sqlite

import (
	"context"
	"github.com/xuanye/one-round/apps/server/internal/domain"
)

func (q *Queries) GetActiveRoundCycle(ctx context.Context, gameSessionID string) (domain.RoundCycle, error) { panic("todo") }
func (q *Queries) CreateRoundCycle(ctx context.Context, rc domain.RoundCycle) error { panic("todo") }
func (q *Queries) ListRoundParticipationStatuses(ctx context.Context, roundCycleID string) ([]domain.RoundParticipationStatus, error) {
	panic("todo")
}
func (q *Queries) UpsertRoundParticipationStatus(ctx context.Context, s domain.RoundParticipationStatus) error {
	panic("todo")
}
func (q *Queries) CompleteRoundCycle(ctx context.Context, roundCycleID string, completedAt string) error { panic("todo") }
```

- [ ] **Step 5: Run migration and focused tests**

Run:

```bash
go test ./tests -run 'Test(RoundStatusCompletesOnlyWhenAllActivePlayersSatisfied|NewRoundOpensOnlyAfterCurrentRoundCompletes|ReversalReopensRoundInsteadOfAdvancing)' -count=1
```

Expected:

- `FAIL` moves from missing schema to missing repository/service logic.

---

### Task 3: Implement Round-Cycle Service and Integrate Normal Score Submission

**Files:**
- Create: `apps/server/internal/app/roundcycle/service.go`
- Modify: `apps/server/internal/app/game/service.go`
- Modify: `apps/server/internal/app/scoretransfer/service.go`
- Modify: `apps/server/internal/infra/sqlite/round_cycle_repository.go`
- Modify: `apps/server/cmd/oneround-server/main.go`

- [ ] **Step 1: Seed round 1 on game creation**

In `apps/server/internal/app/game/service.go`, after creating the owner player:

```go
round := domain.RoundCycle{
	ID:            uuid.NewString(),
	GameSessionID: game.ID,
	RoundNo:       1,
	Status:        domain.RoundCycleStatusActive,
	CreatedAt:     now,
}
if err := q.CreateRoundCycle(ctx, round); err != nil {
	return domain.GameSession{}, err
}
if err := q.UpsertRoundParticipationStatus(ctx, domain.RoundParticipationStatus{
	ID:           uuid.NewString(),
	RoundCycleID: round.ID,
	PlayerID:     ownerPlayer.ID,
	Status:       domain.ParticipationStatusPending,
	UpdatedAt:    now,
}); err != nil {
	return domain.GameSession{}, err
}
```

- [ ] **Step 2: Implement round-cycle service helpers**

Create `apps/server/internal/app/roundcycle/service.go`:

```go
package roundcycle

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/xuanye/one-round/apps/server/internal/domain"
	"github.com/xuanye/one-round/apps/server/internal/infra/sqlite"
)

type Service struct {
	q   *sqlite.Queries
	now func() time.Time
}

func NewService(q *sqlite.Queries, now func() time.Time) *Service {
	return &Service{q: q, now: now}
}

func (s *Service) EnsureActiveRound(ctx context.Context, gameSessionID string) (domain.RoundCycle, error) {
	rc, err := s.q.GetActiveRoundCycle(ctx, gameSessionID)
	if err == nil {
		return rc, nil
	}
	last, err := s.q.GetLatestRoundCycle(ctx, gameSessionID)
	if err != nil {
		return domain.RoundCycle{}, err
	}
	next := domain.RoundCycle{
		ID:            uuid.NewString(),
		GameSessionID: gameSessionID,
		RoundNo:       last.RoundNo + 1,
		Status:        domain.RoundCycleStatusActive,
		CreatedAt:     s.now(),
	}
	if err := s.q.CreateRoundCycle(ctx, next); err != nil {
		return domain.RoundCycle{}, err
	}
	return next, nil
}
```

- [ ] **Step 3: Assign every normal transfer to the active round**

In `apps/server/internal/app/scoretransfer/service.go`, inside the transaction:

```go
round, err := s.roundCycle.EnsureActiveRound(ctx, gameSessionID)
if err != nil {
	return err
}

transfer := domain.ScoreTransfer{
	ID:              transferID,
	GameSessionID:   gameSessionID,
	RoundCycleID:    round.ID,
	SequenceNo:      sequenceNo,
	FromPlayerID:    sender.ID,
	CreatedByUserID: userID,
	IdempotencyKey:  input.IdempotencyKey,
	Amount:          input.Amount,
	Kind:            domain.ScoreTransferKindNormal,
	CreatedAt:       now,
}
```

- [ ] **Step 4: Mark sender and active receivers satisfied for the round**

Still in `scoretransfer/service.go`:

```go
if err := s.roundCycle.MarkParticipantsSatisfied(ctx, round.ID, transfer.ID, append([]string{sender.ID}, input.ReceiverPlayerIDs...)); err != nil {
	return err
}
allSatisfied, err := s.roundCycle.RoundComplete(ctx, round.ID)
if err != nil {
	return err
}
if allSatisfied {
	if err := s.roundCycle.Complete(ctx, round.ID); err != nil {
		return err
	}
}
```

- [ ] **Step 5: Wire the round-cycle service**

In `apps/server/cmd/oneround-server/main.go`:

```go
roundCycleService := roundcycle.NewService(queries, now)
scoreTransferService := scoretransfersvc.NewService(store, queries, gameService, roundCycleService, hub, now)
```

And update the constructor:

```go
func NewService(store *sqlite.Store, q *sqlite.Queries, game *gamesvc.Service, roundCycle *roundcycle.Service, hub realtime.Hub, now func() time.Time) *Service
```

- [ ] **Step 6: Run the normal-round tests**

Run:

```bash
go test ./tests -run 'Test(RoundStatusCompletesOnlyWhenAllActivePlayersSatisfied|NewRoundOpensOnlyAfterCurrentRoundCompletes)' -count=1
```

Expected:

- Both tests `PASS`.
- Reversal test still fails because reversal flow is not implemented yet.

---

### Task 4: Implement Explicit Reversal Flow

**Files:**
- Modify: `apps/server/internal/app/scoretransfer/service.go`
- Modify: `apps/server/internal/infra/sqlite/score_transfer_repository.go`
- Modify: `apps/server/internal/api/dto/score_transfer_dto.go`
- Modify: `apps/server/internal/api/handler/score_transfer_handler.go`

- [ ] **Step 1: Add reversal input/result types**

In `apps/server/internal/app/scoretransfer/service.go`:

```go
type ReverseInput struct {
	IdempotencyKey string
	Reason         string
}

type ReverseResult struct {
	ID         string `json:"id"`
	SequenceNo int    `json:"sequenceNo"`
	Version    int64  `json:"version"`
}
```

- [ ] **Step 2: Add repository helpers for source transfer lookup and reverse marking**

In `apps/server/internal/infra/sqlite/score_transfer_repository.go`:

```go
func (q *Queries) GetScoreTransferForUpdate(ctx context.Context, gameSessionID, transferID string) (domain.ScoreTransfer, error) {
	return q.GetScoreTransfer(ctx, transferID)
}

func (q *Queries) MarkScoreTransferReversed(ctx context.Context, transferID string, reversedAt time.Time) error {
	_, err := q.db.ExecContext(ctx, `UPDATE score_transfers SET reversed_at = ? WHERE id = ?`, encodeTime(reversedAt), transferID)
	return err
}
```

- [ ] **Step 3: Implement `Reverse(...)` transaction**

Add to `scoretransfer/service.go`:

```go
func (s *Service) Reverse(ctx context.Context, userID, gameSessionID, transferID string, input ReverseInput) (ReverseResult, error) {
	if input.IdempotencyKey == "" {
		return ReverseResult{}, domain.ErrIdempotencyKeyRequired
	}

	now := s.now()
	reversalID := uuid.NewString()
	var sequenceNo int
	var version int64

	err := s.store.InTx(ctx, func(q *sqlite.Queries) error {
		original, err := q.GetScoreTransferForUpdate(ctx, gameSessionID, transferID)
		if err != nil {
			return err
		}
		if original.ReversedAt != nil {
			return domain.ErrIdempotencyConflict
		}

		sequenceNo, err = q.NextScoreTransferSequence(ctx, gameSessionID)
		if err != nil {
			return err
		}

		reversalOf := original.ID
		reversal := domain.ScoreTransfer{
			ID:                   reversalID,
			GameSessionID:        gameSessionID,
			RoundCycleID:         original.RoundCycleID,
			SequenceNo:           sequenceNo,
			FromPlayerID:         original.Receivers[0].PlayerID,
			CreatedByUserID:      userID,
			IdempotencyKey:       input.IdempotencyKey,
			Amount:               original.Amount,
			Kind:                 domain.ScoreTransferKindReversal,
			ReversalOfTransferID: &reversalOf,
			CreatedAt:            now,
		}

		if err := q.InsertScoreTransferRaw(ctx, reversal); err != nil {
			return err
		}
		if err := q.MarkScoreTransferReversed(ctx, original.ID, now); err != nil {
			return err
		}
		if err := s.roundCycle.ReopenFromReversal(ctx, original.RoundCycleID, original.ID); err != nil {
			return err
		}
		version, err = q.IncrementGameSessionForTransfer(ctx, gameSessionID, now)
		return err
	})
	if err != nil {
		return ReverseResult{}, err
	}

	return ReverseResult{ID: reversalID, SequenceNo: sequenceNo, Version: version}, nil
}
```

- [ ] **Step 4: Add handler and DTO glue**

In `apps/server/internal/api/handler/score_transfer_handler.go`:

```go
func (h *ScoreTransferHandler) Reverse(w http.ResponseWriter, r *http.Request) {
	var req dto.ReverseScoreTransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, err)
		return
	}
	result, err := h.scoreTransfer.Reverse(
		r.Context(),
		middleware.UserID(r.Context()),
		chi.URLParam(r, "id"),
		chi.URLParam(r, "transferId"),
		scoretransfersvc.ReverseInput{
			IdempotencyKey: req.IdempotencyKey,
			Reason:         req.Reason,
		},
	)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}
```

- [ ] **Step 5: Register the route**

In the router setup:

```go
r.Post("/game-sessions/{id}/score-transfers/{transferId}/reversal", scoreTransferHandler.Reverse)
```

- [ ] **Step 6: Run reversal tests**

Run:

```bash
go test ./tests -run TestReversalReopensRoundInsteadOfAdvancing -count=1
```

Expected:

- `PASS`.

---

### Task 5: Replace Summary Heuristics with Persisted Round Status

**Files:**
- Modify: `apps/server/internal/app/query/summary_service.go`
- Modify: `apps/server/internal/infra/sqlite/round_cycle_repository.go`
- Modify: `apps/server/internal/api/dto/game_dto.go` if shared types are extracted

- [ ] **Step 1: Replace the query model**

In `apps/server/internal/app/query/summary_service.go`:

```go
type RoundStatus struct {
	RoundNo           int      `json:"roundNo"`
	Status            string   `json:"status"`
	PendingPlayerIDs  []string `json:"pendingPlayerIds"`
	PendingPlayerNames []string `json:"pendingPlayerNames"`
	CanStartNextRound bool     `json:"canStartNextRound"`
}
```

- [ ] **Step 2: Remove `CalculateRoundStatus(...)` entirely**

Delete the heuristic function and replace summary loading with:

```go
round, err := s.q.GetLatestRoundCycle(ctx, gameSessionID)
if err != nil {
	return Summary{}, err
}
statuses, err := s.q.ListRoundParticipationStatuses(ctx, round.ID)
if err != nil {
	return Summary{}, err
}

pendingIDs := make([]string, 0)
pendingNames := make([]string, 0)
	nameByID := map[string]string{}
for _, p := range players {
	nameByID[p.ID] = p.DisplayName
}
for _, status := range statuses {
	if status.Status == domain.ParticipationStatusPending {
		pendingIDs = append(pendingIDs, status.PlayerID)
		pendingNames = append(pendingNames, nameByID[status.PlayerID])
	}
}

summary.RoundStatus = RoundStatus{
	RoundNo:            round.RoundNo,
	Status:             string(round.Status),
	PendingPlayerIDs:   pendingIDs,
	PendingPlayerNames: pendingNames,
	CanStartNextRound:  round.Status == domain.RoundCycleStatusComplete,
}
```

- [ ] **Step 3: Run query tests**

Run:

```bash
go test ./internal/app/query -count=1
```

Expected:

- `PASS` with no remaining references to `CalculateRoundStatus`.

---

### Task 6: Update Player Join/Leave Flows to Keep Round Completeness Correct

**Files:**
- Modify: `apps/server/internal/app/player/service.go`
- Modify: `apps/server/internal/app/game/service.go`
- Modify: `apps/server/internal/infra/sqlite/round_cycle_repository.go`
- Modify: `apps/server/tests/round_cycle_test.go`

- [ ] **Step 1: Add the failing join-mid-round test**

Append to `apps/server/tests/round_cycle_test.go`:

```go
func TestJoinerAddedMidRoundStartsPendingInCurrentRound(t *testing.T) {
	ctx := context.Background()
	app := newTestApp(t)
	owner := mustLogin(t, app, "owner-code")
	u2 := mustLogin(t, app, "u2-code")
	u3 := mustLogin(t, app, "u3-code")

	game, _ := mustCreateGameWithOwner(t, app, owner.ID, "家庭局", nil)
	p2 := mustJoinGame(t, app, u2.ID, game.InviteCode, "李四")
	mustSubmitTransfer(t, app, owner.ID, game.ID, []string{p2.ID}, 10, "t1")

	p3 := mustJoinGame(t, app, u3.ID, game.InviteCode, "王五")
	summary, err := app.query.Summary(ctx, owner.ID, game.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(summary.RoundStatus.PendingPlayerIDs) != 1 || summary.RoundStatus.PendingPlayerIDs[0] != p3.ID {
		t.Fatalf("expected joiner pending in current round, got %+v", summary.RoundStatus.PendingPlayerIDs)
	}
}
```

- [ ] **Step 2: Add the failing leave-mid-round test**

Append:

```go
func TestLeavingPlayerBecomesExemptFromCurrentRound(t *testing.T) {
	ctx := context.Background()
	app := newTestApp(t)
	owner := mustLogin(t, app, "owner-code")
	u2 := mustLogin(t, app, "u2-code")
	u3 := mustLogin(t, app, "u3-code")

	game, _ := mustCreateGameWithOwner(t, app, owner.ID, "家庭局", nil)
	p2 := mustJoinGame(t, app, u2.ID, game.InviteCode, "李四")
	_ = mustJoinGame(t, app, u3.ID, game.InviteCode, "王五")

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
}
```

- [ ] **Step 3: Update join flow**

In `apps/server/internal/app/player/service.go`, after join/reactivate succeeds:

```go
round, err := q.GetActiveRoundCycle(ctx, gameSessionID)
if err == nil {
	_ = q.UpsertRoundParticipationStatus(ctx, domain.RoundParticipationStatus{
		ID:           uuid.NewString(),
		RoundCycleID: round.ID,
		PlayerID:     player.ID,
		Status:       domain.ParticipationStatusPending,
		UpdatedAt:    now,
	})
}
```

- [ ] **Step 4: Update leave flow**

In the leave transaction:

```go
round, err := q.GetActiveRoundCycle(ctx, gameSessionID)
if err == nil {
	if err := q.SetRoundParticipationStatus(ctx, round.ID, leavingPlayer.ID, domain.ParticipationStatusExempt, now); err != nil {
		return err
	}
	complete, err := s.roundCycle.RoundComplete(ctx, round.ID)
	if err != nil {
		return err
	}
	if complete {
		if err := s.roundCycle.Complete(ctx, round.ID); err != nil {
			return err
		}
	}
}
```

- [ ] **Step 5: Run join/leave round tests**

Run:

```bash
go test ./tests -run 'Test(JoinerAddedMidRoundStartsPendingInCurrentRound|LeavingPlayerBecomesExemptFromCurrentRound)' -count=1
```

Expected:

- Both tests `PASS`.

---

### Task 7: Mini Program Contract Update and Warning UX

**Files:**
- Modify: `apps/miniprogram/src/models/game-session.ts`
- Modify: `apps/miniprogram/src/services/score.service.ts`
- Modify: `apps/miniprogram/src/pages/game-detail/index.ts`
- Modify: `apps/miniprogram/src/pages/game-detail/index.wxml`
- Modify: `apps/miniprogram/src/pages/game-detail/index.wxss`
- Modify: `apps/miniprogram/src/pages/score-input/index.ts`

- [ ] **Step 1: Update the summary model**

In `apps/miniprogram/src/models/game-session.ts`:

```ts
export type RoundStatus = {
  roundNo: number;
  status: 'active' | 'complete';
  pendingPlayerIds: string[];
  pendingPlayerNames: string[];
  canStartNextRound: boolean;
};
```

- [ ] **Step 2: Update detail-page warning logic**

In `apps/miniprogram/src/pages/game-detail/index.ts`:

```ts
const roundStatus = summary.roundStatus || null;
const pendingNamesText = roundStatus?.pendingPlayerNames?.join('、') || '';

this.setData({
  roundStatus,
  uninvolvedNamesText: pendingNamesText,
});
```

And in `inputScore()`:

```ts
if (roundStatus && roundStatus.status === 'active' && roundStatus.pendingPlayerNames.length > 0) {
  const mePending = roundStatus.pendingPlayerIds.includes(me.id);
  if (!mePending) {
    wx.showModal({
      title: '提示',
      content: `当前第 ${roundStatus.roundNo} 轮还有 ${roundStatus.pendingPlayerNames.join('、')} 尚未完成计分参与。确认继续新计分吗？`,
      confirmText: '继续',
      cancelText: '取消',
      success: (res) => {
        if (res.confirm) {
          wx.navigateTo({ url: `/pages/score-input/index?id=${this.data.id}` });
        }
      }
    });
    return;
  }
}
```

- [ ] **Step 3: Update the banner wording**

In `apps/miniprogram/src/pages/game-detail/index.wxml`:

```xml
<view wx:if="{{roundStatus && roundStatus.status === 'active' && roundStatus.pendingPlayerNames.length > 0}}" class="round-warning-banner card">
  <view class="round-warning-text">
    <text class="iconfont inline-icon animate-pulse" style="margin-right: 8rpx; color: #ba1a1a;">{{icons.info}}</text>
    第{{roundStatus.roundNo}}轮还有 {{uninvolvedNamesText}} 尚未完成计分参与。
  </view>
</view>
```

- [ ] **Step 4: Add reversal API wrapper**

In `apps/miniprogram/src/services/score.service.ts`:

```ts
export function reverseScoreTransfer(gameSessionId: string, transferId: string, idempotencyKey: string, reason: string) {
  return request({
    url: `/api/game-sessions/${gameSessionId}/score-transfers/${transferId}/reversal`,
    method: 'POST',
    data: { idempotencyKey, reason },
  });
}
```

- [ ] **Step 5: Run Mini Program type-check**

Run:

```bash
npm run check
```

Expected:

- `PASS`.

---

### Task 8: Final Verification, Docs, and Release Notes

**Files:**
- Modify: `docs/requirements/game-scoring.md`
- Optional modify: `apps/miniprogram/README.md`

- [ ] **Step 1: Update the requirements doc with the new explicit correction model**

Append to `docs/requirements/game-scoring.md`:

```md
- 轮次是服务端持久化状态，不由客户端或查询层从历史计分临时推断。
- 计分错误修正使用显式 reversal 接口，不依赖用户手工输入“反向计分”让系统猜测。
- 每条计分转移都归属于一个持久化 round cycle。
- 当前轮提醒以服务端返回的 pending players 为准。
```

- [ ] **Step 2: Run the server suite**

Run:

```bash
go test ./...
```

Expected:

- All server tests `PASS`.

- [ ] **Step 3: Run Mini Program checks**

Run:

```bash
npm run check
npm test
```

Expected:

- TypeScript check `PASS`.
- Existing Mini Program script tests `PASS`.

- [ ] **Step 4: Smoke-check changed files only**

Run:

```bash
git diff -- apps/server apps/miniprogram docs/requirements/game-scoring.md
```

Expected:

- Only round-cycle, reversal, summary, and Mini Program warning changes appear.
- No unrelated refactors.

---

## Self-Review

Spec coverage:

- Round reminder correctness is implemented by persisted `round_cycles` and `round_participation_statuses`.
- Explicit correction semantics are implemented by reversal API instead of heuristic reverse-transfer guessing.
- Active join/leave behavior is covered so reminders remain correct during mutable player membership.
- Mini Program warning UX is covered and now depends only on server DTO state.

Placeholder scan:

- No `TODO`, `TBD`, or "handle appropriately" placeholders remain.
- Every task lists exact files and concrete commands.

Type consistency:

- Backend uses `RoundCycle`, `RoundParticipationStatus`, `ScoreTransferKind`, and `ReverseInput/Result`.
- Summary DTO consistently uses `roundNo`, `status`, `pendingPlayerIds`, `pendingPlayerNames`, `canStartNextRound`.
- Mini Program model matches the server DTO names.

Commit policy:

- This repository says not to commit unless explicitly instructed. This plan intentionally omits commit commands. If the user later asks for commits, commit only after Task 8 verification passes.

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-06-21-round-cycle-stateful-scoring.md`. Two execution options:

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

Which approach?
