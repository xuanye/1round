# Game Scoring Server Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement all server-side behavior in `docs/requirements/game-scoring.md`: current game lifecycle, invite join, score transfers, participant profile updates, settlement, history, public settlement share, realtime invalidation, and scheduled auto settlement.

**Architecture:** Keep transport in `internal/api`, orchestration in `internal/app`, domain types in `internal/domain`, and all SQLite access under `internal/infra/sqlite`. HTTP summary and detail APIs are authoritative; WebSocket events remain small invalidation signals. Replace the legacy `Round` model with `ScoreTransfer` and remove optional `zeroSumRequired` behavior.

**Tech Stack:** Go 1.24, chi, database/sql, modernc.org/sqlite, goose migrations, JWT auth, nhooyr websocket, slog, `go test ./...`.

---

## Important Scope Notes

- This is one server-wide plan because the requirements are coupled through shared persistence and lifecycle rules.
- Do not implement Mini Program pages in this plan.
- Do not preserve `/rounds` compatibility. The requirement explicitly says to use `/score-transfers`.
- Do not edit existing migrations `00001` through `00005`; add append-only migrations.
- Keep generated SQLite data files under `apps/server/data/` untouched unless tests intentionally create disposable databases.
- This is a breaking API change for the Mini Program client. Server and Mini Program must be released together, or a temporary compatibility branch must be planned separately. This server plan intentionally removes `/rounds` in Task 9 after `/score-transfers` is implemented and tested.

## Target API Surface

Authenticated APIs:

- `POST /api/game-sessions`
- `GET /api/game-sessions/current`
- `GET /api/game-sessions/{id}`
- `GET /api/game-sessions/{id}/summary`
- `GET /api/game-sessions/{id}/ranking`
- `POST /api/game-sessions/join-preview`
- `POST /api/game-sessions/join`
- `POST /api/game-sessions/{id}/leave`
- `POST /api/game-sessions/{id}/finish`
- `POST /api/game-sessions/{id}/finish-requests`
- `POST /api/game-sessions/{id}/finish-requests/{requestId}/approve`
- `POST /api/game-sessions/{id}/finish-requests/{requestId}/reject`
- `PATCH /api/game-sessions/{id}/my-profile`
- `POST /api/game-sessions/{id}/score-transfers`
- `GET /api/game-sessions/{id}/score-transfers`
- `GET /api/history/game-sessions`
- `GET /api/history/game-sessions/{id}`

Public API:

- `GET /api/public/settlements/{shareToken}`

WebSocket:

- `GET /ws/game-sessions/{id}`
- Events are invalidation-only: `game.updated`, `participant.joined`, `participant.left`, `participant.updated`, `score_transfer.submitted`, `finish_request.updated`, `game.finished`, `game.voided`.

## Explicit Non-Goals

- Do not implement score transfer edit, delete, or undo. The requirement says: `第一版不做撤销或编辑计分功能`.
- Do not keep `zeroSumRequired` as a business option. The requirement says all games are zero-sum.
- Do not add durable state to WebSocket events. HTTP summary/detail endpoints remain the source of truth.

## HTTP Error Mapping

Update `apps/server/internal/api/response/response.go` as new domain errors are added:

| Domain error | HTTP status | API code | Notes |
| --- | ---: | ---: | --- |
| `ErrInvalidArgument` | `400` | `40001` | Bad request payload or invalid capacity/name |
| `ErrInvalidPlayer` | `400` | `40001` | Unknown participant, self receiver, duplicate receiver, wrong game |
| `ErrInvalidScoreTransferAmount` | `400` | `40001` | Amount must be positive integer |
| `ErrScoreTransferReceiverRequired` | `400` | `40001` | At least one receiver is required |
| `ErrIdempotencyKeyRequired` | `400` | `40001` | Score transfer submission must include idempotency key |
| `ErrUnauthorized` | `401` | `40101` | Missing or invalid JWT |
| `ErrForbidden` | `403` | `40301` | Generic forbidden |
| `ErrGameMemberRequired` | `403` | `40301` | User has no historical membership where required |
| `ErrParticipantRequired` | `403` | `40301` | User is not a current active participant |
| `ErrOwnerRequired` | `403` | `40301` | Only current owner can perform action |
| `ErrNotFound` | `404` | `40401` | Missing game, participant, transfer, or finish request |
| `ErrPublicShareUnavailable` | `404` | `40401` | Share token missing, voided, or not settled |
| `ErrConflict` | `409` | `40901` | Generic state conflict |
| `ErrGameSessionFinished` | `409` | `40901` | Mutating settled game |
| `ErrGameSessionVoided` | `409` | `40901` | Mutating voided game |
| `ErrActiveGameExists` | `409` | `40901` | User already has a current game |
| `ErrGameCapacityFull` | `409` | `40901` | Active participant count reached capacity |
| `ErrCannotLeaveWithNonZeroScore` | `409` | `40901` | Participant score is not zero |
| `ErrParticipantInactive` | `409` | `40901` | Receiver or actor is no longer active |
| `ErrDuplicateDisplayName` | `409` | `40901` | Historical display name uniqueness violation |
| `ErrIdempotencyConflict` | `409` | `40901` | Same idempotency key with different payload |
| `ErrFinishRequestPending` | `409` | `40901` | Another pending finish request exists |
| `ErrFinishRequestNotPending` | `409` | `40901` | Request was already handled |

## Realtime Event Contract

WebSocket events are invalidation signals. Clients must refetch HTTP summary/detail after receiving them.

| Event | When emitted | Payload |
| --- | --- | --- |
| `game.updated` | Game metadata or owner changes | `{}` |
| `participant.joined` | Active participant added or reactivated | `{"playerId":"..."}` |
| `participant.left` | Active participant leaves and game remains active | `{"playerId":"..."}` |
| `participant.updated` | Participant display profile changes | `{"playerId":"..."}` |
| `score_transfer.submitted` | Score transfer persisted and totals updated | `{"sequenceNo":1}` |
| `finish_request.updated` | Finish request created, approved, or rejected | `{"requestId":"...","status":"pending"}` |
| `game.finished` | Manual or automatic settlement succeeds | `{}` |
| `game.voided` | Unscored game is voided | `{}` |

Every event must include the existing envelope fields:

```go
type Event struct {
	Type          string    `json:"type"`
	GameSessionID string    `json:"gameSessionId"`
	Version       int64     `json:"version"`
	Payload       any       `json:"payload,omitempty"`
	SentAt        time.Time `json:"sentAt"`
}
```

Do not add `score_transfer.edited` or `score_transfer.deleted`; edit/delete is out of scope for v1.

## Concurrency Strategy

- SQLite is configured with `SetMaxOpenConns(1)`, so server writes are serialized through one database connection.
- Multi-table mutations must run inside `Store.InTx`.
- Score transfer sequence numbers are allocated from the transaction's current `game_sessions.round_count + 1`.
- `UNIQUE(game_session_id, sequence_no)` protects sequence integrity if connection settings change later.
- `UNIQUE(game_session_id, created_by_user_id, idempotency_key)` makes repeated client submissions idempotent.
- `game_sessions.version` is a client invalidation/version marker, not an optimistic lock. Do not describe it as conflict detection unless implementation adds `WHERE version = ?`.
- Concurrent valid score transfers may both succeed; final order is the server commit order.
- If `sqlite.Open` is later changed to allow multiple writer connections, `InsertScoreTransfer` must catch `UNIQUE(game_session_id, sequence_no)` conflicts and retry sequence allocation inside a new transaction. Do not increase `MaxOpenConns` without adding that retry path and a concurrent submission regression test.

## Test Helper Evolution

Keep `apps/server/tests/service_test.go` helpers synchronized with each service introduced by the tasks.

After Task 5, `testApp` should have this shape:

```go
type testApp struct {
	auth          *authsvc.Service
	game          *gamesvc.Service
	player        *playersvc.Service
	scoreTransfer *scoretransfersvc.Service
	settlement    *settlementsvc.Service
	query         *querysvc.Service
	hub           *realtime.MemoryHub
	db            *sql.DB
	now           func() time.Time
	setNow        func(time.Time)
}
```

Task-specific helper changes:

- Task 2: replace `round` expectations with participant-aware game/query helpers.
- Task 3: update `createGame` to pass `maxParticipants *int`.
- Task 5: add `scoreTransfer` service and remove new test usage of `round`.
- Task 6: add `settlement` service and route handlers that call it.
- Task 8: make the test clock mutable; do not rebuild services per test.

Use one mutable clock closure in `newTestApp`:

```go
func newTestApp(t *testing.T) *testApp {
	t.Helper()
	current := time.Date(2026, 6, 19, 0, 0, 0, 0, time.UTC)
	now := func() time.Time { return current }
	setNow := func(next time.Time) { current = next }

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
	tokens := jwtauth.NewJWTService("test-signing-key", 720*time.Hour)
	gameService := gamesvc.NewService(store, q, hub, now)
	settlementService := settlementsvc.NewService(store, q, hub, now)
	scoreTransferService := scoretransfersvc.NewService(store, q, gameService, hub, now)

	return &testApp{
		auth: authsvc.NewService(q, wechat.FakeClient{}, tokens, now),
		game: gameService,
		player: playersvc.NewService(q, gameService, hub, now),
		scoreTransfer: scoreTransferService,
		settlement: settlementService,
		query: querysvc.NewService(q, gameService),
		hub: hub,
		db: db,
		now: now,
		setNow: setNow,
	}
}
```

If constructor dependencies differ after implementation, update this helper in the same task that changes the constructor. Do not leave tests with parallel helper variants.

## File Structure

Create:

- `apps/server/migrations/00006_game_scoring_server.sql`  
  Adds lifecycle columns, participant state, score transfer tables, finish request table, settlement share token, and indexes.
- `apps/server/internal/domain/score_transfer.go`  
  Canonical score transfer domain types.
- `apps/server/internal/domain/finish_request.go`  
  End-game request domain types.
- `apps/server/internal/app/scoretransfer/service.go`  
  Score transfer use-case and idempotency.
- `apps/server/internal/app/settlement/service.go`  
  Manual finish, automatic settlement, voiding, history/public read orchestration.
- `apps/server/internal/app/scheduler/auto_settlement.go`  
  Background ticker wrapper around settlement service.
- `apps/server/internal/api/dto/score_transfer_dto.go`  
  Score transfer request/response DTOs.
- `apps/server/internal/api/dto/history_dto.go`  
  History and public settlement DTOs if query structs are not sufficient.
- `apps/server/internal/api/handler/score_transfer_handler.go`  
  `/score-transfers` handlers.
- `apps/server/internal/api/handler/history_handler.go`  
  Authenticated history and public settlement handlers.
- `apps/server/internal/infra/sqlite/score_transfer_repository.go`  
  Score transfer persistence and paginated reads.
- `apps/server/internal/infra/sqlite/settlement_repository.go`  
  Settlement, void, current game, finish request, history, public share persistence.

Modify:

- `apps/server/internal/domain/game_session.go`  
  Remove `ZeroSumRequired`; add `voided`, settlement, capacity, activity, and share fields.
- `apps/server/internal/domain/player.go`  
  Rename semantics to participant where exposed by new APIs; add active/left and join-order data.
- `apps/server/internal/domain/errors.go`  
  Add explicit errors for active game conflict, capacity full, cannot leave non-zero score, duplicate display name, finish request conflict, idempotency conflict, and public share unavailable.
- `apps/server/internal/app/game/service.go`  
  Current game lookup, create with capacity, join preview/join by invite, owner transfer, direct finish delegation.
- `apps/server/internal/app/player/service.go`  
  Rework into participant profile update/leave semantics; no hard delete for historical participants.
- `apps/server/internal/app/query/summary_service.go`  
  Summary/ranking/detail/history projections from score transfers and active participant state.
- `apps/server/internal/api/dto/game_dto.go`
- `apps/server/internal/api/dto/player_dto.go`
- `apps/server/internal/api/handler/game_handler.go`
- `apps/server/internal/api/handler/player_handler.go`
- `apps/server/internal/api/router.go`
- `apps/server/internal/realtime/event.go`
- `apps/server/cmd/oneround-server/main.go`
- `apps/server/tests/service_test.go`
- `apps/server/tests/api_test.go`

Remove after replacement:

- `apps/server/internal/domain/round.go`
- `apps/server/internal/app/round/service.go`
- `apps/server/internal/api/dto/round_dto.go`
- `apps/server/internal/api/handler/round_handler.go`
- `apps/server/internal/infra/sqlite/round_repository.go`

---

## Data Model

### New Migration Shape

Add `apps/server/migrations/00006_game_scoring_server.sql`:

```sql
-- +goose Up
ALTER TABLE game_sessions ADD COLUMN max_participants INTEGER NULL;
ALTER TABLE game_sessions ADD COLUMN settled_at TEXT NULL;
ALTER TABLE game_sessions ADD COLUMN voided_at TEXT NULL;
ALTER TABLE game_sessions ADD COLUMN last_scored_at TEXT NULL;
ALTER TABLE game_sessions ADD COLUMN public_share_token TEXT NULL;

CREATE UNIQUE INDEX idx_game_sessions_public_share_token
ON game_sessions(public_share_token)
WHERE public_share_token IS NOT NULL;

ALTER TABLE players ADD COLUMN active INTEGER NOT NULL DEFAULT 1;
ALTER TABLE players ADD COLUMN joined_order INTEGER NOT NULL DEFAULT 0;
ALTER TABLE players ADD COLUMN left_at TEXT NULL;

CREATE UNIQUE INDEX idx_players_game_display_name
ON players(game_session_id, display_name);

CREATE UNIQUE INDEX idx_players_game_user
ON players(game_session_id, user_id)
WHERE user_id IS NOT NULL;

CREATE INDEX idx_players_game_active_order
ON players(game_session_id, active, joined_order);

CREATE TABLE score_transfers (
    id TEXT PRIMARY KEY,
    game_session_id TEXT NOT NULL,
    sequence_no INTEGER NOT NULL,
    from_player_id TEXT NOT NULL,
    created_by_user_id TEXT NOT NULL,
    idempotency_key TEXT NOT NULL,
    amount INTEGER NOT NULL,
    created_at TEXT NOT NULL,
    FOREIGN KEY(game_session_id) REFERENCES game_sessions(id),
    FOREIGN KEY(from_player_id) REFERENCES players(id),
    FOREIGN KEY(created_by_user_id) REFERENCES users(id),
    UNIQUE(game_session_id, sequence_no),
    UNIQUE(game_session_id, created_by_user_id, idempotency_key),
    CHECK(amount > 0)
);

CREATE INDEX idx_score_transfers_game_sequence
ON score_transfers(game_session_id, sequence_no DESC);

CREATE TABLE score_transfer_receivers (
    id TEXT PRIMARY KEY,
    score_transfer_id TEXT NOT NULL,
    player_id TEXT NOT NULL,
    receiver_order INTEGER NOT NULL,
    FOREIGN KEY(score_transfer_id) REFERENCES score_transfers(id),
    FOREIGN KEY(player_id) REFERENCES players(id),
    UNIQUE(score_transfer_id, player_id)
);

CREATE INDEX idx_score_transfer_receivers_transfer_order
ON score_transfer_receivers(score_transfer_id, receiver_order);

CREATE INDEX idx_score_transfer_receivers_player
ON score_transfer_receivers(player_id);

CREATE TABLE finish_requests (
    id TEXT PRIMARY KEY,
    game_session_id TEXT NOT NULL,
    requested_by_player_id TEXT NOT NULL,
    status TEXT NOT NULL,
    created_at TEXT NOT NULL,
    decided_at TEXT NULL,
    decided_by_player_id TEXT NULL,
    FOREIGN KEY(game_session_id) REFERENCES game_sessions(id),
    FOREIGN KEY(requested_by_player_id) REFERENCES players(id),
    FOREIGN KEY(decided_by_player_id) REFERENCES players(id)
);

CREATE UNIQUE INDEX idx_finish_requests_one_pending
ON finish_requests(game_session_id)
WHERE status = 'pending';

CREATE INDEX idx_finish_requests_game_created
ON finish_requests(game_session_id, created_at DESC);

-- Backfill existing rows.
UPDATE game_sessions
SET last_scored_at = CASE WHEN round_count > 0 THEN updated_at ELSE NULL END;

UPDATE players
SET joined_order = (
    SELECT COUNT(*)
    FROM players p2
    WHERE p2.game_session_id = players.game_session_id
      AND (p2.created_at < players.created_at OR (p2.created_at = players.created_at AND p2.id <= players.id))
);

-- +goose Down
DROP INDEX idx_finish_requests_game_created;
DROP INDEX idx_finish_requests_one_pending;
DROP TABLE finish_requests;
DROP INDEX idx_score_transfer_receivers_player;
DROP INDEX idx_score_transfer_receivers_transfer_order;
DROP TABLE score_transfer_receivers;
DROP INDEX idx_score_transfers_game_sequence;
DROP TABLE score_transfers;
DROP INDEX idx_players_game_active_order;
DROP INDEX idx_players_game_user;
DROP INDEX idx_players_game_display_name;
DROP INDEX idx_game_sessions_public_share_token;
```

`Down` cannot remove SQLite columns without rebuilding tables. That is acceptable for local rollback only if documented in code review; production migration direction is append-only.

---

## Task 1: Domain and Error Vocabulary

**Files:**

- Create: `apps/server/internal/domain/score_transfer.go`
- Create: `apps/server/internal/domain/finish_request.go`
- Modify: `apps/server/internal/domain/game_session.go`
- Modify: `apps/server/internal/domain/player.go`
- Modify: `apps/server/internal/domain/errors.go`
- Test: `apps/server/tests/service_test.go`

- [ ] **Step 1: Write failing domain validation tests**

Add tests:

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
cd apps/server
go test ./tests -run TestScoreTransferRequestValidation -count=1
```

Expected: fail because `ValidateScoreTransferInput` and new errors do not exist.

- [ ] **Step 3: Add domain types**

Add:

```go
package domain

import "time"

type ScoreTransfer struct {
	ID              string
	GameSessionID   string
	SequenceNo      int
	FromPlayerID    string
	CreatedByUserID string
	IdempotencyKey  string
	Amount          int
	CreatedAt       time.Time
	Receivers       []ScoreTransferReceiver
}

type ScoreTransferReceiver struct {
	ID              string
	ScoreTransferID string
	PlayerID        string
	ReceiverOrder   int
}

func ValidateScoreTransferInput(amount int, receiverPlayerIDs []string) error {
	if amount <= 0 {
		return ErrInvalidScoreTransferAmount
	}
	if len(receiverPlayerIDs) == 0 {
		return ErrScoreTransferReceiverRequired
	}
	seen := map[string]struct{}{}
	for _, id := range receiverPlayerIDs {
		if id == "" {
			return ErrInvalidPlayer
		}
		if _, ok := seen[id]; ok {
			return ErrInvalidPlayer
		}
		seen[id] = struct{}{}
	}
	return nil
}
```

Add:

```go
package domain

import "time"

type FinishRequestStatus string

const (
	FinishRequestStatusPending  FinishRequestStatus = "pending"
	FinishRequestStatusApproved FinishRequestStatus = "approved"
	FinishRequestStatusRejected FinishRequestStatus = "rejected"
)

type FinishRequest struct {
	ID                  string
	GameSessionID       string
	RequestedByPlayerID string
	Status              FinishRequestStatus
	CreatedAt           time.Time
	DecidedAt           *time.Time
	DecidedByPlayerID   *string
}
```

Update `GameSessionStatus`:

```go
const (
	GameSessionStatusActive   GameSessionStatus = "active"
	GameSessionStatusFinished GameSessionStatus = "finished"
	GameSessionStatusVoided   GameSessionStatus = "voided"
)
```

Update `GameSession`:

```go
type GameSession struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	InviteCode       string            `json:"inviteCode"`
	OwnerUserID      string            `json:"ownerUserId"`
	Status           GameSessionStatus `json:"status"`
	MaxParticipants  *int              `json:"maxParticipants"`
	ScoreTransferCnt int               `json:"scoreTransferCount"`
	Version          int64             `json:"version"`
	PublicShareToken *string           `json:"publicShareToken,omitempty"`
	LastScoredAt     *time.Time        `json:"lastScoredAt,omitempty"`
	SettledAt        *time.Time        `json:"settledAt,omitempty"`
	VoidedAt         *time.Time        `json:"voidedAt,omitempty"`
	CreatedAt        time.Time         `json:"createdAt"`
	UpdatedAt        time.Time         `json:"updatedAt"`
}
```

Keep database column `round_count` for now, but map it to `ScoreTransferCnt` in repositories. Do not expose `zeroSumRequired`.

Update `Player`:

```go
type Player struct {
	ID            string     `json:"id"`
	GameSessionID string     `json:"gameSessionId"`
	UserID        *string    `json:"userId"`
	DisplayName   string     `json:"displayName"`
	TotalScore    int        `json:"totalScore"`
	Active        bool       `json:"active"`
	JoinedOrder   int        `json:"joinedOrder"`
	LeftAt        *time.Time `json:"leftAt,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}
```

Add errors:

```go
ErrActiveGameExists              = errors.New("active game exists")
ErrGameSessionVoided            = errors.New("game session voided")
ErrGameCapacityFull             = errors.New("game capacity full")
ErrCannotLeaveWithNonZeroScore  = errors.New("cannot leave with non-zero score")
ErrParticipantRequired          = errors.New("participant required")
ErrParticipantInactive          = errors.New("participant inactive")
ErrDuplicateDisplayName         = errors.New("duplicate display name")
ErrInvalidScoreTransferAmount   = errors.New("invalid score transfer amount")
ErrScoreTransferReceiverRequired = errors.New("score transfer receiver required")
ErrIdempotencyKeyRequired       = errors.New("idempotency key required")
ErrIdempotencyConflict          = errors.New("idempotency conflict")
ErrFinishRequestPending         = errors.New("finish request pending")
ErrFinishRequestNotPending      = errors.New("finish request not pending")
ErrOwnerRequired                = errors.New("owner required")
ErrPublicShareUnavailable      = errors.New("public share unavailable")
```

- [ ] **Step 4: Run test to verify it passes**

Run:

```bash
cd apps/server
go test ./tests -run TestScoreTransferRequestValidation -count=1
```

Expected: pass.

- [ ] **Step 5: Commit**

```bash
git add apps/server/internal/domain apps/server/tests/service_test.go
git commit -m "feat(server): add game scoring domain vocabulary"
```

---

## Task 2: SQLite Migration and Repository Mapping

**Files:**

- Create: `apps/server/migrations/00006_game_scoring_server.sql`
- Create: `apps/server/internal/infra/sqlite/score_transfer_repository.go`
- Create: `apps/server/internal/infra/sqlite/settlement_repository.go`
- Modify: `apps/server/internal/infra/sqlite/game_repository.go`
- Modify: `apps/server/internal/infra/sqlite/player_repository.go`
- Test: `apps/server/tests/service_test.go`

- [ ] **Step 1: Write failing repository integration tests**

Add tests:

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
cd apps/server
go test ./tests -run 'Test(CreateGameStoresCapacityAndOwnerParticipant|CurrentGameExcludesFinishedAndVoidedGames)' -count=1
```

Expected: fail because `Create` signature, `Current`, and `ActiveParticipants` do not exist.

- [ ] **Step 3: Add migration**

Use the SQL from the **Data Model** section exactly.

- [ ] **Step 4: Update repository scans**

In `game_repository.go`, select and scan:

```sql
SELECT id, name, invite_code, owner_user_id, status, max_participants,
       round_count, version, public_share_token, last_scored_at,
       settled_at, voided_at, created_at, updated_at
FROM game_sessions
```

Use nullable helpers:

```go
func nullStringPtr(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	return &v.String
}

func nullIntPtr(v sql.NullInt64) *int {
	if !v.Valid {
		return nil
	}
	n := int(v.Int64)
	return &n
}

func nullTimePtr(v sql.NullString) (*time.Time, error) {
	if !v.Valid {
		return nil, nil
	}
	t, err := decodeTime(v.String)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
```

Add repository methods:

```go
func (q *Queries) GetCurrentGameForUser(ctx context.Context, userID string) (*domain.GameSession, error)
func (q *Queries) CountActiveParticipants(ctx context.Context, gameSessionID string) (int, error)
func (q *Queries) NextJoinedOrder(ctx context.Context, gameSessionID string) (int, error)
func (q *Queries) GetActivePlayerByUser(ctx context.Context, gameSessionID, userID string) (domain.Player, error)
func (q *Queries) GetHistoricalPlayerByUser(ctx context.Context, gameSessionID, userID string) (domain.Player, error)
func (q *Queries) ListActivePlayers(ctx context.Context, gameSessionID string) ([]domain.Player, error)
func (q *Queries) ListHistoricalPlayers(ctx context.Context, gameSessionID string) ([]domain.Player, error)
```

Query rules:

```sql
-- current game
SELECT gs...
FROM game_sessions gs
JOIN players p ON p.game_session_id = gs.id
WHERE p.user_id = ?
  AND p.active = 1
  AND gs.status = 'active'
ORDER BY gs.updated_at DESC
LIMIT 1;

-- active participants
SELECT ...
FROM players
WHERE game_session_id = ? AND active = 1
ORDER BY joined_order ASC;

-- historical participants
SELECT ...
FROM players
WHERE game_session_id = ?
ORDER BY joined_order ASC;
```

- [ ] **Step 5: Update tests helpers**

Change helper signature:

```go
func createGame(t *testing.T, app *testApp, userID string, maxParticipants *int) domain.GameSession {
	t.Helper()
	game, err := app.game.Create(context.Background(), userID, "家庭聚会", maxParticipants)
	if err != nil {
		t.Fatal(err)
	}
	return game
}
```

- [ ] **Step 6: Run focused tests**

Run:

```bash
cd apps/server
go test ./tests -run 'Test(CreateGameStoresCapacityAndOwnerParticipant|CurrentGameExcludesFinishedAndVoidedGames)' -count=1
```

Expected: pass.

- [ ] **Step 7: Commit**

```bash
git add apps/server/migrations/00006_game_scoring_server.sql apps/server/internal/infra/sqlite apps/server/tests/service_test.go
git commit -m "feat(server): add game scoring persistence"
```

---

## Task 3: Game Creation, Current Game, Invite Preview, Join, Profile Update

**Files:**

- Modify: `apps/server/internal/app/game/service.go`
- Modify: `apps/server/internal/app/player/service.go`
- Modify: `apps/server/internal/app/query/summary_service.go`
- Modify: `apps/server/internal/api/dto/game_dto.go`
- Modify: `apps/server/internal/api/dto/player_dto.go`
- Modify: `apps/server/internal/api/handler/game_handler.go`
- Modify: `apps/server/internal/api/handler/player_handler.go`
- Modify: `apps/server/internal/api/router.go`
- Test: `apps/server/tests/service_test.go`
- Test: `apps/server/tests/api_test.go`

- [ ] **Step 1: Write failing service tests**

Add tests:

```go
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
```

- [ ] **Step 2: Run service tests to verify failure**

Run:

```bash
cd apps/server
go test ./tests -run 'Test(UserCanHaveOnlyOneCurrentGame|JoinPreviewDoesNotCreateParticipant|JoinEnforcesCapacityAndDisplayNameUniqueness)' -count=1
```

Expected: fail because new join/profile behavior is missing.

- [ ] **Step 3: Implement service behavior**

Change create signature:

```go
func (s *Service) Create(ctx context.Context, userID, name string, maxParticipants *int) (domain.GameSession, error)
```

Validation:

```go
if maxParticipants != nil && (*maxParticipants < 2 || *maxParticipants > 10) {
	return domain.GameSession{}, domain.ErrInvalidArgument
}
current, err := s.q.GetCurrentGameForUser(ctx, userID)
if err != nil {
	return domain.GameSession{}, err
}
if current != nil {
	return domain.GameSession{}, domain.ErrActiveGameExists
}
```

Owner participant creation in the same transaction:

```go
ownerPlayer := domain.Player{
	ID: uuid.NewString(), GameSessionID: session.ID, UserID: &userID,
	DisplayName: defaultDisplayName(userID), TotalScore: 0,
	Active: true, JoinedOrder: 1, CreatedAt: now, UpdatedAt: now,
}
```

Default name:

```go
func defaultDisplayName(userID string) string {
	if len(userID) >= 2 {
		return "老书记" + userID[len(userID)-2:]
	}
	return "老书记00"
}
```

Add:

```go
type JoinPreview struct {
	GameSessionID          string          `json:"gameSessionId"`
	Name                   string          `json:"name"`
	OwnerDisplayName       string          `json:"ownerDisplayName"`
	ParticipantCount       int             `json:"participantCount"`
	MaxParticipants        *int            `json:"maxParticipants"`
	Participants           []PlayerPreview `json:"participants"`
	CurrentUserDisplayName string          `json:"currentUserDisplayName"`
	AlreadyJoined          bool            `json:"alreadyJoined"`
}

type PlayerPreview struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
}
```

Join rules:

- Find active session by invite code.
- Reject finished/voided games.
- If user has another current game, return `ErrActiveGameExists`.
- If user already active in this game, return the game id.
- If user was historical inactive in this game, reactivate same player with score `0`.
- Capacity counts only active participants.
- Display name must be unique across all historical participants in this game, excluding the same reactivated player.
- Broadcast `participant.joined`.

Profile update:

```go
func (s *Service) UpdateMyProfile(ctx context.Context, userID, gameSessionID, displayName string) (domain.Player, error)
```

Rules:

- User must be historical participant.
- Game must be active.
- Name unique among all historical participants except self.
- Broadcast `participant.updated`.

- [ ] **Step 4: Write failing API test**

Add:

```go
func TestCurrentPreviewJoinAndProfileAPI(t *testing.T) {
	app := newTestApp(t)
	tokens := jwtauth.NewJWTService("test-signing-key", 720*time.Hour)
	router := api.NewRouter(slog.Default(), api.Services{
		Auth: app.auth, Game: app.game, Player: app.player, Query: app.query,
		Tokens: tokens, WebSocket: wshandler.NewWebSocketHandler(app.game, app.hub, 4, time.Second),
	})

	ownerToken := loginHTTP(t, router, "owner-code")
	game := postJSON[map[string]any](t, router, ownerToken, "/api/game-sessions", map[string]any{"name": "家庭聚会", "maxParticipants": 2})
	current := getJSON[map[string]any](t, router, ownerToken, "/api/game-sessions/current")
	if current["id"] != game["id"] {
		t.Fatalf("unexpected current game: %+v", current)
	}

	joinerToken := loginHTTP(t, router, "joiner-code")
	preview := postJSON[map[string]any](t, router, joinerToken, "/api/game-sessions/join-preview", map[string]any{"inviteCode": game["inviteCode"]})
	if preview["participantCount"].(float64) != 1 {
		t.Fatalf("unexpected preview: %+v", preview)
	}
	joined := postJSON[map[string]any](t, router, joinerToken, "/api/game-sessions/join", map[string]any{"inviteCode": game["inviteCode"], "displayName": "妈妈"})
	if joined["gameSessionId"] != game["id"] {
		t.Fatalf("unexpected join: %+v", joined)
	}
	updated := patchJSON[map[string]any](t, router, joinerToken, "/api/game-sessions/"+game["id"].(string)+"/my-profile", map[string]any{"displayName": "阿姨"})
	if updated["displayName"] != "阿姨" {
		t.Fatalf("unexpected profile: %+v", updated)
	}
}
```

Add `patchJSON` helper mirroring `postJSON`.

- [ ] **Step 5: Update router and handlers**

DTOs:

```go
type CreateGameRequest struct {
	Name            string `json:"name"`
	MaxParticipants *int   `json:"maxParticipants"`
}

type JoinGameRequest struct {
	InviteCode  string `json:"inviteCode"`
	DisplayName string `json:"displayName"`
}
```

Routes:

```go
r.Get("/game-sessions/current", gameHandler.Current)
r.Post("/game-sessions/join-preview", gameHandler.JoinPreview)
r.Patch("/game-sessions/{id}/my-profile", playerHandler.UpdateMyProfile)
```

- [ ] **Step 6: Run focused tests**

Run:

```bash
cd apps/server
go test ./tests -run 'Test(UserCanHaveOnlyOneCurrentGame|JoinPreviewDoesNotCreateParticipant|JoinEnforcesCapacityAndDisplayNameUniqueness|CurrentPreviewJoinAndProfileAPI)' -count=1
```

Expected: pass.

- [ ] **Step 7: Commit**

```bash
git add apps/server/internal/app apps/server/internal/api apps/server/internal/infra/sqlite apps/server/tests
git commit -m "feat(server): implement current game and invite join"
```

---

## Task 4: Leave Game, Owner Transfer, Void, Auto-Finish-on-Last-Leave

**Files:**

- Modify: `apps/server/internal/app/player/service.go`
- Modify: `apps/server/internal/app/game/service.go`
- Modify: `apps/server/internal/infra/sqlite/player_repository.go`
- Modify: `apps/server/internal/infra/sqlite/settlement_repository.go`
- Modify: `apps/server/internal/api/handler/game_handler.go`
- Modify: `apps/server/internal/api/router.go`
- Test: `apps/server/tests/service_test.go`
- Test: `apps/server/tests/api_test.go`

- [ ] **Step 1: Write failing service tests**

Add:

```go
func TestLeaveRequiresZeroScore(t *testing.T) {
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
		ReceiverPlayerIDs: []string{joinerPlayer.ID}, Amount: 10, IdempotencyKey: "k1",
	})
	if err != nil {
		t.Fatal(err)
	}

	err = app.player.Leave(ctx, owner, game.ID)
	if err != domain.ErrCannotLeaveWithNonZeroScore {
		t.Fatalf("expected non-zero leave error, got %v", err)
	}
	if ownerPlayer.ID == "" {
		t.Fatal("owner participant missing")
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
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
cd apps/server
go test ./tests -run 'Test(LeaveRequiresZeroScore|LastZeroScoreParticipantLeaveVoidsUnscoredGame)' -count=1
```

Expected: fail because leave/void behavior is missing.

- [ ] **Step 3: Implement leave flow**

Service method:

```go
func (s *Service) Leave(ctx context.Context, userID, gameSessionID string) error
```

Rules inside one transaction:

- Load active session.
- Load active player for user.
- If `TotalScore != 0`, return `ErrCannotLeaveWithNonZeroScore`.
- Mark player inactive and set `left_at`.
- If leaving owner, transfer `owner_user_id` to active player with smallest `joined_order`.
- If no active players remain:
  - `ScoreTransferCnt == 0`: set status `voided`, `voided_at`, increment version, broadcast `game.voided`.
  - `ScoreTransferCnt > 0`: finish game, set `settled_at`, `public_share_token`, increment version, broadcast `game.finished`.
- If active players remain: increment version and broadcast `participant.left`.

Repository updates:

```sql
UPDATE players
SET active = 0, left_at = ?, updated_at = ?
WHERE game_session_id = ? AND user_id = ? AND active = 1;
```

Owner transfer query:

```sql
SELECT user_id
FROM players
WHERE game_session_id = ? AND active = 1 AND user_id IS NOT NULL
ORDER BY joined_order ASC
LIMIT 1;
```

- [ ] **Step 4: Add API route**

Route:

```go
r.Post("/game-sessions/{id}/leave", gameHandler.Leave)
```

Handler:

```go
func (h *GameHandler) Leave(w http.ResponseWriter, r *http.Request) {
	err := h.player.Leave(r.Context(), middleware.UserID(r.Context()), chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]bool{"left": true})
}
```

If `GameHandler` does not have `player`, either inject it or put the handler on `PlayerHandler`; keep route under game sessions.

- [ ] **Step 5: Run tests**

Run:

```bash
cd apps/server
go test ./tests -run 'Test(LeaveRequiresZeroScore|LastZeroScoreParticipantLeaveVoidsUnscoredGame)' -count=1
```

Expected: pass.

- [ ] **Step 6: Commit**

```bash
git add apps/server/internal/app apps/server/internal/api apps/server/internal/infra/sqlite apps/server/tests
git commit -m "feat(server): implement participant leave lifecycle"
```

---

## Task 5: Score Transfers and Paginated Details

**Files:**

- Create: `apps/server/internal/app/scoretransfer/service.go`
- Create: `apps/server/internal/api/dto/score_transfer_dto.go`
- Create: `apps/server/internal/api/handler/score_transfer_handler.go`
- Create: `apps/server/internal/infra/sqlite/score_transfer_repository.go`
- Modify: `apps/server/internal/app/query/summary_service.go`
- Modify: `apps/server/internal/api/router.go`
- Modify: `apps/server/internal/realtime/event.go`
- Modify: `apps/server/cmd/oneround-server/main.go`
- Test: `apps/server/tests/service_test.go`
- Test: `apps/server/tests/api_test.go`

- [ ] **Step 1: Write failing score transfer tests**

Add:

```go
func TestScoreTransferDebitsActorAndCreditsReceivers(t *testing.T) {
	app := newTestApp(t)
	ctx := context.Background()
	owner := login(t, app, "owner-code")
	u2 := login(t, app, "u2-code")
	u3 := login(t, app, "u3-code")
	game := createGame(t, app, owner, nil)
	if _, err := app.game.Join(ctx, u2, game.InviteCode, "妈妈"); err != nil { t.Fatal(err) }
	if _, err := app.game.Join(ctx, u3, game.InviteCode, "孩子"); err != nil { t.Fatal(err) }
	p1, _ := app.query.MyParticipant(ctx, owner, game.ID)
	p2, _ := app.query.MyParticipant(ctx, u2, game.ID)
	p3, _ := app.query.MyParticipant(ctx, u3, game.ID)

	result, err := app.scoreTransfer.Submit(ctx, owner, game.ID, scoretransfersvc.SubmitInput{
		ReceiverPlayerIDs: []string{p2.ID, p3.ID},
		Amount: 20,
		IdempotencyKey: "submit-1",
	})
	if err != nil { t.Fatal(err) }
	if result.SequenceNo != 1 || result.Version != 2 {
		t.Fatalf("unexpected result: %+v", result)
	}
	summary, err := app.query.Summary(ctx, owner, game.ID)
	if err != nil { t.Fatal(err) }
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
	if _, err := app.game.Join(ctx, joiner, game.InviteCode, "妈妈"); err != nil { t.Fatal(err) }
	p2, _ := app.query.MyParticipant(ctx, joiner, game.ID)
	input := scoretransfersvc.SubmitInput{ReceiverPlayerIDs: []string{p2.ID}, Amount: 10, IdempotencyKey: "same-key"}

	first, err := app.scoreTransfer.Submit(ctx, owner, game.ID, input)
	if err != nil { t.Fatal(err) }
	second, err := app.scoreTransfer.Submit(ctx, owner, game.ID, input)
	if err != nil { t.Fatal(err) }
	if first.ID != second.ID || first.Version != second.Version {
		t.Fatalf("expected same idempotent result: first=%+v second=%+v", first, second)
	}
	summary, _ := app.query.Summary(ctx, owner, game.ID)
	if summary.ScoreTransferCount != 1 {
		t.Fatalf("expected one transfer, got %+v", summary)
	}
}

func TestScoreTransferRejectsSelfAndInactiveReceiver(t *testing.T) {
	app := newTestApp(t)
	ctx := context.Background()
	owner := login(t, app, "owner-code")
	joiner := login(t, app, "joiner-code")
	game := createGame(t, app, owner, nil)
	if _, err := app.game.Join(ctx, joiner, game.InviteCode, "妈妈"); err != nil { t.Fatal(err) }
	ownerPlayer, _ := app.query.MyParticipant(ctx, owner, game.ID)
	joinerPlayer, _ := app.query.MyParticipant(ctx, joiner, game.ID)

	_, err := app.scoreTransfer.Submit(ctx, owner, game.ID, scoretransfersvc.SubmitInput{
		ReceiverPlayerIDs: []string{ownerPlayer.ID}, Amount: 10, IdempotencyKey: "self",
	})
	if err != domain.ErrInvalidPlayer {
		t.Fatalf("expected invalid self receiver, got %v", err)
	}
	if err := app.player.Leave(ctx, joiner, game.ID); err != nil { t.Fatal(err) }
	_, err = app.scoreTransfer.Submit(ctx, owner, game.ID, scoretransfersvc.SubmitInput{
		ReceiverPlayerIDs: []string{joinerPlayer.ID}, Amount: 10, IdempotencyKey: "inactive",
	})
	if err != domain.ErrParticipantInactive {
		t.Fatalf("expected inactive receiver error, got %v", err)
	}
}
```

- [ ] **Step 2: Run tests to verify failure**

Run:

```bash
cd apps/server
go test ./tests -run 'TestScoreTransfer' -count=1
```

Expected: fail because score transfer service is missing.

- [ ] **Step 3: Implement score transfer service**

Types:

```go
type SubmitInput struct {
	ReceiverPlayerIDs []string
	Amount            int
	IdempotencyKey    string
}

type SubmitResult struct {
	ID         string `json:"id"`
	SequenceNo int    `json:"sequenceNo"`
	Version    int64  `json:"version"`
}
```

Algorithm:

1. Validate amount, receivers, idempotency key.
2. Load active game; reject finished/voided.
3. Load active sender by `userID`; sender must be current participant.
4. Reject any receiver equal to sender.
5. Load all receiver players as active in the same game.
6. If an idempotent transfer exists for `(gameSessionID, createdByUserID, idempotencyKey)` with same receiver set and amount, return previous result.
7. If idempotency key exists with different payload, return `ErrIdempotencyConflict`.
8. In one transaction:
   - Insert `score_transfers`.
   - Insert receivers using joined order.
   - `UPDATE players SET total_score = total_score - amount * receiver_count WHERE id = from_player_id`.
   - `UPDATE players SET total_score = total_score + amount WHERE id IN receivers`.
   - Increment `round_count`, version, updated_at, and `last_scored_at`.
9. Broadcast `score_transfer.submitted`.

- [ ] **Step 4: Implement repository methods**

Required methods:

```go
func (q *Queries) GetScoreTransferByIdempotencyKey(ctx context.Context, gameSessionID, userID, key string) (domain.ScoreTransfer, error)
func (q *Queries) InsertScoreTransfer(ctx context.Context, transfer domain.ScoreTransfer, nextVersion int64) error
func (q *Queries) ListScoreTransfers(ctx context.Context, gameSessionID string, beforeSequenceNo *int, limit int) ([]domain.ScoreTransfer, error)
func (q *Queries) GetScoreTransferCount(ctx context.Context, gameSessionID string) (int, error)
```

Pagination query:

```sql
SELECT id, game_session_id, sequence_no, from_player_id, created_by_user_id,
       idempotency_key, amount, created_at
FROM score_transfers
WHERE game_session_id = ?
  AND (? IS NULL OR sequence_no < ?)
ORDER BY sequence_no DESC
LIMIT ?;
```

- [ ] **Step 5: Update query projections**

Summary must expose:

```go
type ScoreTransferSummary struct {
	ID          string    `json:"id"`
	SequenceNo  int       `json:"sequenceNo"`
	FromPlayerID string    `json:"fromPlayerId"`
	ReceiverIDs []string  `json:"receiverPlayerIds"`
	Amount      int       `json:"amount"`
	CreatedAt   time.Time `json:"createdAt"`
	Text        string    `json:"text"`
}
```

Text format:

```go
func formatTransferText(from string, receivers []string, amount int) string {
	if len(receivers) == 1 {
		return fmt.Sprintf("%s 给 %s +%d", from, receivers[0], amount)
	}
	return fmt.Sprintf("%s 给 %s 各 +%d", from, strings.Join(receivers, "、"), amount)
}
```

Summary player ordering:

- Main summary: active participants by joined order.
- Ranking: active participants by `total_score DESC, joined_order ASC`.
- Detail/history/settlement: historical participants by final score desc for results; transfers desc.

- [ ] **Step 6: Add API**

Request DTO:

```go
type SubmitScoreTransferRequest struct {
	ReceiverPlayerIDs []string `json:"receiverPlayerIds"`
	Amount            int      `json:"amount"`
	IdempotencyKey    string   `json:"idempotencyKey"`
}
```

Routes:

```go
r.Post("/game-sessions/{id}/score-transfers", scoreTransferHandler.Submit)
r.Get("/game-sessions/{id}/score-transfers", scoreTransferHandler.List)
```

Query params:

- `beforeSequenceNo`
- `limit`, default `20`, max `100`

- [ ] **Step 7: Update wiring**

In `main.go` and tests:

```go
scoreTransferService := scoretransfersvc.NewService(store, queries, gameService, hub, now)
```

`api.Services` replaces `Round` with `ScoreTransfer`.

- [ ] **Step 8: Run score transfer tests**

Run:

```bash
cd apps/server
go test ./tests -run 'TestScoreTransfer' -count=1
```

Expected: pass.

- [ ] **Step 9: Commit**

```bash
git add apps/server/internal/app/scoretransfer apps/server/internal/api apps/server/internal/infra/sqlite apps/server/internal/realtime apps/server/cmd/oneround-server apps/server/tests
git commit -m "feat(server): implement score transfers"
```

---

## Task 6: Finish Requests and Manual Settlement

**Files:**

- Create: `apps/server/internal/app/settlement/service.go`
- Modify: `apps/server/internal/app/game/service.go`
- Modify: `apps/server/internal/infra/sqlite/settlement_repository.go`
- Modify: `apps/server/internal/api/handler/game_handler.go`
- Modify: `apps/server/internal/api/router.go`
- Modify: `apps/server/internal/realtime/event.go`
- Test: `apps/server/tests/service_test.go`
- Test: `apps/server/tests/api_test.go`

- [ ] **Step 1: Write failing finish request tests**

Add:

```go
func TestOwnerCanFinishDirectlyAndFreezeGame(t *testing.T) {
	app := newTestApp(t)
	ctx := context.Background()
	owner := login(t, app, "owner-code")
	joiner := login(t, app, "joiner-code")
	game := createGame(t, app, owner, nil)
	if _, err := app.game.Join(ctx, joiner, game.InviteCode, "妈妈"); err != nil { t.Fatal(err) }

	finished, err := app.settlement.FinishDirect(ctx, owner, game.ID)
	if err != nil { t.Fatal(err) }
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
	if _, err := app.game.Join(ctx, joiner, game.InviteCode, "妈妈"); err != nil { t.Fatal(err) }

	req, err := app.settlement.RequestFinish(ctx, joiner, game.ID)
	if err != nil { t.Fatal(err) }
	if req.Status != domain.FinishRequestStatusPending {
		t.Fatalf("unexpected request: %+v", req)
	}
	req, err = app.settlement.RejectFinishRequest(ctx, owner, game.ID, req.ID)
	if err != nil { t.Fatal(err) }
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
	if _, err := app.game.Join(ctx, u2, game.InviteCode, "妈妈"); err != nil { t.Fatal(err) }
	if _, err := app.game.Join(ctx, u3, game.InviteCode, "孩子"); err != nil { t.Fatal(err) }
	if _, err := app.settlement.RequestFinish(ctx, u2, game.ID); err != nil { t.Fatal(err) }
	_, err := app.settlement.RequestFinish(ctx, u3, game.ID)
	if err != domain.ErrFinishRequestPending {
		t.Fatalf("expected pending request conflict, got %v", err)
	}
}
```

- [ ] **Step 2: Run tests to verify failure**

Run:

```bash
cd apps/server
go test ./tests -run 'Test(OwnerCanFinishDirectlyAndFreezeGame|NonOwnerCreatesFinishRequestAndOwnerCanReject|OnlyOnePendingFinishRequest)' -count=1
```

Expected: fail because settlement service is missing.

- [ ] **Step 3: Implement settlement service**

Constructor:

```go
func NewService(store *sqlite.Store, q *sqlite.Queries, hub realtime.Hub, now func() time.Time) *Service
```

Methods:

```go
func (s *Service) FinishDirect(ctx context.Context, userID, gameSessionID string) (domain.GameSession, error)
func (s *Service) RequestFinish(ctx context.Context, userID, gameSessionID string) (domain.FinishRequest, error)
func (s *Service) ApproveFinishRequest(ctx context.Context, userID, gameSessionID, requestID string) (domain.GameSession, error)
func (s *Service) RejectFinishRequest(ctx context.Context, userID, gameSessionID, requestID string) (domain.FinishRequest, error)
```

Rules:

- Owner can direct finish active game.
- Non-owner creates pending request.
- Owner approving request finishes game.
- Owner rejecting request marks request rejected.
- Request status remains pending across score transfers and owner transfer.
- All finish mutations increment game version and broadcast invalidation.
- Finishing sets `settled_at` and `public_share_token`; voided games never get share tokens.

Share token:

```go
func GenerateShareToken() (string, error) {
	buf := make([]byte, 18)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
```

- [ ] **Step 4: Update `game.Finish`**

Make `game.Finish` call `settlement.FinishDirect` or remove direct finish from game service and route handler. Do not keep duplicate finish logic.

- [ ] **Step 5: Add API routes**

Routes:

```go
r.Post("/game-sessions/{id}/finish", gameHandler.Finish)
r.Post("/game-sessions/{id}/finish-requests", gameHandler.RequestFinish)
r.Post("/game-sessions/{id}/finish-requests/{requestId}/approve", gameHandler.ApproveFinishRequest)
r.Post("/game-sessions/{id}/finish-requests/{requestId}/reject", gameHandler.RejectFinishRequest)
```

Handler behavior:

- `Finish`: owner direct finish; non-owner should call request endpoint. If keeping old button compatibility, return `ErrOwnerRequired` for non-owner.
- `RequestFinish`: active participant only.
- Approve/reject: owner only.

- [ ] **Step 6: Run tests**

Run:

```bash
cd apps/server
go test ./tests -run 'Test(OwnerCanFinishDirectlyAndFreezeGame|NonOwnerCreatesFinishRequestAndOwnerCanReject|OnlyOnePendingFinishRequest)' -count=1
```

Expected: pass.

- [ ] **Step 7: Commit**

```bash
git add apps/server/internal/app/settlement apps/server/internal/app/game apps/server/internal/api apps/server/internal/infra/sqlite apps/server/internal/realtime apps/server/tests
git commit -m "feat(server): implement finish requests and settlement"
```

---

## Task 7: History, Settlement Detail, Public Settlement Share

**Files:**

- Create: `apps/server/internal/api/handler/history_handler.go`
- Create: `apps/server/internal/api/dto/history_dto.go`
- Modify: `apps/server/internal/app/query/summary_service.go`
- Modify: `apps/server/internal/infra/sqlite/settlement_repository.go`
- Modify: `apps/server/internal/api/router.go`
- Test: `apps/server/tests/api_test.go`
- Test: `apps/server/tests/service_test.go`

- [ ] **Step 1: Write failing history tests**

Add:

```go
func TestHistoryOnlyListsSettledGamesForParticipant(t *testing.T) {
	app := newTestApp(t)
	ctx := context.Background()
	owner := login(t, app, "owner-code")
	other := login(t, app, "other-code")
	game := createGame(t, app, owner, nil)
	if _, err := app.settlement.FinishDirect(ctx, owner, game.ID); err != nil { t.Fatal(err) }

	ownerHistory, err := app.query.History(ctx, owner, nil, 20)
	if err != nil { t.Fatal(err) }
	if len(ownerHistory.Items) != 1 || ownerHistory.Items[0].ID != game.ID {
		t.Fatalf("unexpected owner history: %+v", ownerHistory)
	}
	otherHistory, err := app.query.History(ctx, other, nil, 20)
	if err != nil { t.Fatal(err) }
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
	if err != nil { t.Fatal(err) }

	share, err := app.query.PublicSettlement(ctx, *finished.PublicShareToken)
	if err != nil { t.Fatal(err) }
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
```

- [ ] **Step 2: Run tests to verify failure**

Run:

```bash
cd apps/server
go test ./tests -run 'Test(HistoryOnlyListsSettledGamesForParticipant|PublicSettlementShareOmitsAvatarsAndTransferDetails)' -count=1
```

Expected: fail because history/public query methods are missing.

- [ ] **Step 3: Implement query DTOs**

Types:

```go
type HistoryPage struct {
	Items      []HistoryItem `json:"items"`
	NextCursor *string       `json:"nextCursor,omitempty"`
}

type HistoryItem struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	SettledAt          time.Time `json:"settledAt"`
	ScoreTransferCount int       `json:"scoreTransferCount"`
	MyFinalScore        int       `json:"myFinalScore"`
}

type SettlementDetail struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	SettledAt      time.Time              `json:"settledAt"`
	Participants   []SettlementParticipant `json:"participants"`
	ScoreTransfers []ScoreTransferSummary `json:"scoreTransfers"`
	NextCursor     *int                   `json:"nextCursor,omitempty"`
}

type SettlementParticipant struct {
	ID          string  `json:"id"`
	DisplayName string  `json:"displayName"`
	AvatarURL   *string `json:"avatarUrl,omitempty"`
	FinalScore  int     `json:"finalScore"`
}

type PublicSettlement struct {
	GameSessionID string                  `json:"gameSessionId"`
	Name          string                  `json:"name"`
	SettledAt     time.Time               `json:"settledAt"`
	Participants  []SettlementParticipant `json:"participants"`
	ScoreTransfers []ScoreTransferSummary `json:"scoreTransfers"`
}
```

For public settlement, force:

```go
participant.AvatarURL = nil
scoreTransfers = nil
```

- [ ] **Step 4: Implement query methods**

```go
func (s *Service) History(ctx context.Context, userID string, beforeSettledAt *time.Time, limit int) (HistoryPage, error)
func (s *Service) SettlementDetail(ctx context.Context, userID, gameSessionID string, beforeSequenceNo *int, limit int) (SettlementDetail, error)
func (s *Service) PublicSettlement(ctx context.Context, shareToken string) (PublicSettlement, error)
```

Rules:

- History lists only `status = 'finished'`.
- History requires current user has a historical player row in the game.
- Voided games are excluded.
- Detail requires historical participation.
- Detail participants include active and inactive historical participants.
- Detail participants sort by final score desc, joined order asc.
- Detail score transfers are paginated desc.
- Public share is available only for finished games with matching token.
- Public share displays name, settlement date, display names, final scores; no avatars; no score transfer details.
- Public share tokens have no v1 rotation or revocation endpoint. Treat the URL as bearer access; if a token leaks, the only v1 mitigation is operational intervention, not an end-user revoke action.

- [ ] **Step 5: Add API handlers and routes**

Routes:

```go
r.Get("/history/game-sessions", historyHandler.List)
r.Get("/history/game-sessions/{id}", historyHandler.Detail)
```

Public route outside auth group:

```go
r.Get("/api/public/settlements/{shareToken}", historyHandler.PublicSettlement)
```

Pagination:

- History: `beforeSettledAt` RFC3339 string, `limit`.
- Detail transfers: `beforeSequenceNo`, `limit`.

- [ ] **Step 6: Run tests**

Run:

```bash
cd apps/server
go test ./tests -run 'Test(HistoryOnlyListsSettledGamesForParticipant|PublicSettlementShareOmitsAvatarsAndTransferDetails)' -count=1
```

Expected: pass.

- [ ] **Step 7: Commit**

```bash
git add apps/server/internal/app/query apps/server/internal/api apps/server/internal/infra/sqlite apps/server/tests
git commit -m "feat(server): add history and public settlement share"
```

---

## Task 8: Scheduled Auto Settlement and Auto Void

**Files:**

- Create: `apps/server/internal/app/scheduler/auto_settlement.go`
- Modify: `apps/server/internal/app/settlement/service.go`
- Modify: `apps/server/internal/infra/sqlite/settlement_repository.go`
- Modify: `apps/server/internal/config/config.go`
- Modify: `apps/server/config.example.yaml`
- Modify: `apps/server/cmd/oneround-server/main.go`
- Test: `apps/server/tests/service_test.go`

- [ ] **Step 1: Write failing auto-settlement tests**

Add:

```go
func TestAutoVoidUnscoredGameAfter24Hours(t *testing.T) {
	app := newTestAppAt(t, time.Date(2026, 6, 19, 0, 0, 0, 0, time.UTC))
	ctx := context.Background()
	user := login(t, app, "owner-code")
	game := createGame(t, app, user, nil)

	app.setNow(time.Date(2026, 6, 20, 0, 1, 0, 0, time.UTC))
	result, err := app.settlement.SettleInactiveGames(ctx, 24*time.Hour)
	if err != nil { t.Fatal(err) }
	if result.Voided != 1 || result.Finished != 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
	g, err := app.game.GetForHistoricalMember(ctx, user, game.ID)
	if err != nil { t.Fatal(err) }
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
	if _, err := app.game.Join(ctx, joiner, game.InviteCode, "妈妈"); err != nil { t.Fatal(err) }
	p2, _ := app.query.MyParticipant(ctx, joiner, game.ID)
	if _, err := app.scoreTransfer.Submit(ctx, owner, game.ID, scoretransfersvc.SubmitInput{
		ReceiverPlayerIDs: []string{p2.ID}, Amount: 10, IdempotencyKey: "k1",
	}); err != nil { t.Fatal(err) }

	app.setNow(time.Date(2026, 6, 20, 0, 1, 0, 0, time.UTC))
	result, err := app.settlement.SettleInactiveGames(ctx, 24*time.Hour)
	if err != nil { t.Fatal(err) }
	if result.Finished != 1 || result.Voided != 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
	g, err := app.game.GetForHistoricalMember(ctx, owner, game.ID)
	if err != nil { t.Fatal(err) }
	if g.Status != domain.GameSessionStatusFinished || g.SettledAt == nil {
		t.Fatalf("expected finished, got %+v", g)
	}
}
```

Add helpers:

```go
func newTestAppAt(t *testing.T, initial time.Time) *testApp {
	t.Helper()
	app := newTestApp(t)
	app.setNow(initial)
	return app
}
```

Use the mutable `now`/`setNow` closure from **Test Helper Evolution**. Do not create a second service setup path for clock-based tests.

- [ ] **Step 2: Run tests to verify failure**

Run:

```bash
cd apps/server
go test ./tests -run 'TestAuto(Void|Finish)' -count=1
```

Expected: fail because auto settlement is missing.

- [ ] **Step 3: Implement settlement scan**

Types:

```go
type InactiveSettlementResult struct {
	Finished int `json:"finished"`
	Voided   int `json:"voided"`
}
```

Method:

```go
func (s *Service) SettleInactiveGames(ctx context.Context, threshold time.Duration) (InactiveSettlementResult, error)
```

Repository query:

```sql
SELECT id
FROM game_sessions
WHERE status = 'active'
  AND (
    (round_count = 0 AND created_at <= ?)
    OR
    (round_count > 0 AND last_scored_at IS NOT NULL AND last_scored_at <= ?)
  )
ORDER BY created_at ASC
LIMIT 100;
```

For each candidate, perform idempotent transaction:

- Re-read game by id.
- If no longer active, skip.
- If `ScoreTransferCnt == 0`, void.
- Else finish and create share token.
- Broadcast corresponding event after commit.

Boundary behavior:

- Join, leave, display-name changes, and finish-request actions do not refresh auto-settlement timers.
- A game with joins but no score transfers still has `round_count = 0` and `last_scored_at = NULL`; it is auto-voided based only on `created_at`.
- A game with at least one score transfer is auto-finished based only on `last_scored_at`.
- Pending finish requests do not delay automatic finish or void.

- [ ] **Step 4: Add scheduler**

`auto_settlement.go`:

```go
type AutoSettlementRunner struct {
	service  *settlement.Service
	interval time.Duration
	threshold time.Duration
	logger   *slog.Logger
}

func (r *AutoSettlementRunner) Start(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				result, err := r.service.SettleInactiveGames(ctx, r.threshold)
				if err != nil {
					r.logger.Error("auto settlement failed", "error", err)
					continue
				}
				if result.Finished > 0 || result.Voided > 0 {
					r.logger.Info("auto settlement completed", "finished", result.Finished, "voided", result.Voided)
				}
			}
		}
	}()
}
```

Config:

```yaml
settlement:
  autoCheckIntervalSeconds: 300
  inactivityHours: 24
```

Defaults:

- check interval: `300s`
- threshold: `24h`

- [ ] **Step 5: Wire in main**

After services are created:

```go
runner := scheduler.NewAutoSettlementRunner(settlementService, logger, cfg.Settlement.AutoCheckInterval(), cfg.Settlement.InactivityThreshold())
runner.Start(ctx)
```

- [ ] **Step 6: Run tests**

Run:

```bash
cd apps/server
go test ./tests -run 'TestAuto(Void|Finish)' -count=1
```

Expected: pass.

- [ ] **Step 7: Commit**

```bash
git add apps/server/internal/app/scheduler apps/server/internal/app/settlement apps/server/internal/infra/sqlite apps/server/internal/config apps/server/config.example.yaml apps/server/cmd/oneround-server apps/server/tests
git commit -m "feat(server): add scheduled auto settlement"
```

---

## Task 9: Remove Legacy Round API and Finish Contract Cleanup

**Files:**

- Delete: `apps/server/internal/domain/round.go`
- Delete: `apps/server/internal/app/round/service.go`
- Delete: `apps/server/internal/api/dto/round_dto.go`
- Delete: `apps/server/internal/api/handler/round_handler.go`
- Delete: `apps/server/internal/infra/sqlite/round_repository.go`
- Modify: `apps/server/internal/api/router.go`
- Modify: `apps/server/cmd/oneround-server/main.go`
- Modify: `apps/server/tests/service_test.go`
- Modify: `apps/server/tests/api_test.go`
- Modify: `README.md`

- [ ] **Step 1: Search for legacy names**

Run:

```bash
rg '\bRound\b|rounds|RoundSubmitted|zeroSumRequired|ZeroSumRequired' apps/server README.md docs -g '!docs/superpowers/plans/2026-06-19-game-scoring-server.md'
```

Expected: matches still exist before cleanup.

- [ ] **Step 2: Remove legacy routes and services**

Remove:

```go
r.Post("/game-sessions/{id}/rounds", roundHandler.Submit)
r.Get("/game-sessions/{id}/rounds/recent", roundHandler.Recent)
```

Remove `Round` from `api.Services`, `main.go`, `testApp`, and router setup.

- [ ] **Step 3: Update README WebSocket/API examples**

Replace `/rounds` with `/score-transfers` and `round.submitted` with `score_transfer.submitted`.

- [ ] **Step 4: Run legacy search again**

Run:

```bash
rg '\bRound\b|rounds|RoundSubmitted|zeroSumRequired|ZeroSumRequired' apps/server README.md docs -g '!docs/superpowers/plans/2026-06-19-game-scoring-server.md'
```

Expected: no matches except old migrations `00004_create_rounds.sql` and references inside migration comments if kept. Existing migration table names can remain because migrations are historical.

- [ ] **Step 5: Run full server tests**

Run:

```bash
cd apps/server
go test ./...
```

Expected: pass.

- [ ] **Step 6: Commit**

```bash
git add README.md apps/server
git commit -m "refactor(server): remove legacy round API"
```

---

## Task 10: Full API Regression Coverage

**Files:**

- Modify: `apps/server/tests/api_test.go`
- Modify: `apps/server/tests/service_test.go`

- [ ] **Step 1: Add end-to-end API scenario**

Add one API test covering:

1. Owner login.
2. Create game with capacity.
3. Current game returns created game.
4. Joiner logs in.
5. Join preview shows no score details.
6. Joiner confirms join.
7. Owner submits multi-receiver score transfer.
8. Summary shows participants by join order and transfer details by newest first.
9. Ranking shows score desc.
10. Non-owner creates finish request.
11. Owner approves.
12. History lists finished game for participants.
13. Public settlement share works without auth.
14. Further join/leave/score transfer/profile update all fail.

Core assertion snippets:

```go
summary := getJSON[map[string]any](t, router, ownerToken, "/api/game-sessions/"+gameID+"/summary")
participants := summary["players"].([]any)
if len(participants) != 2 {
	t.Fatalf("unexpected participants: %+v", participants)
}
transfers := summary["scoreTransfers"].([]any)
if len(transfers) != 1 {
	t.Fatalf("unexpected transfers: %+v", transfers)
}

history := getJSON[map[string]any](t, router, ownerToken, "/api/history/game-sessions")
items := history["items"].([]any)
if len(items) != 1 {
	t.Fatalf("unexpected history: %+v", history)
}

public := getJSON[map[string]any](t, router, "", "/api/public/settlements/"+shareToken)
if public["gameSessionId"] != gameID {
	t.Fatalf("unexpected public share: %+v", public)
}
```

Add `postJSONExpectStatus` helper for failure assertions:

```go
func postJSONExpectStatus(t *testing.T, router http.Handler, token, path string, payload any, want int) {
	t.Helper()
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != want {
		t.Fatalf("POST %s status %d, want %d body %s", path, rec.Code, want, rec.Body.String())
	}
}
```

- [ ] **Step 2: Run full tests**

Run:

```bash
cd apps/server
go test ./...
```

Expected: pass.

- [ ] **Step 3: Run formatting**

Run:

```bash
cd apps/server
gofmt -w cmd internal tests
go test ./...
```

Expected: pass.

- [ ] **Step 4: Commit**

```bash
git add apps/server/tests
git commit -m "test(server): cover game scoring API lifecycle"
```

---

## Final Verification

Run:

```bash
cd apps/server
go test ./...
```

Expected:

```text
ok  	github.com/xuanye/one-round/apps/server/tests
```

Run:

```bash
rg '\bRound\b|rounds|RoundSubmitted|zeroSumRequired|ZeroSumRequired' apps/server README.md docs -g '!docs/superpowers/plans/2026-06-19-game-scoring-server.md'
```

Expected:

- No service/API/domain matches.
- Historical migration `00004_create_rounds.sql` may still match because migrations are append-only.

Run:

```bash
git status --short
```

Expected: clean after planned commits.

---

## Self-Review

### Spec Coverage

- Login and identity: covered through existing auth plus default game display name and profile update.
- Current game: covered by `Current`, create conflict, leave/finish/void exclusion.
- Create game: covered by capacity, owner participant creation, no `zeroSumRequired`.
- Invite and join: covered by preview, join by invite code, capacity, duplicate active game, reactivation, no raw game id in invite flow.
- Participants: covered by active/historical state, joined order, active-only summary/ranking.
- Exit: covered by zero-score leave, inactive historical participants, owner transfer, last participant void/finish.
- Scoring: covered by `ScoreTransfer`, positive amount, multi-receiver debit/credit, idempotency, no self, active participant validation.
- Score details: covered by paginated score transfers and text formatting.
- Main summary/ranking: covered by authoritative summary and ranking projections.
- Finish game: covered by owner direct finish, non-owner requests, approve/reject, one pending request.
- Settlement: covered by finished read-only state, share token, final score projections.
- Auto settlement and void: covered by scheduler plus idempotent settlement scan.
- History: covered by participant-only finished history and detail.
- Public settlement share: covered by unauthenticated public endpoint omitting avatars and transfer details.
- Realtime: covered by invalidation event constants and broadcasts.

### Known Implementation Risks

- The migration keeps legacy `round_count` column and old `rounds` tables for append-only compatibility; code should treat `round_count` as `score_transfer_count`.
- SQLite cannot drop added columns in `Down`; production rollback should use backups, not destructive column rebuilds.
- Existing Mini Program client will break until updated from `/rounds` to `/score-transfers`.
- There is no v1 public share token revocation or rotation mechanism. If a settlement URL is leaked, access remains available until an operator changes data or a future feature adds revocation.
- Public share token must be treated as bearer access; keep it high entropy and do not expose it for active/voided games.
