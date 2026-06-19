package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/xuanye/one-round/apps/server/internal/domain"
)

func (q *Queries) CreateFinishRequest(ctx context.Context, r domain.FinishRequest) error {
	_, err := q.db.ExecContext(ctx, `INSERT INTO finish_requests (id, game_session_id, requested_by_player_id, status, created_at) VALUES (?, ?, ?, ?, ?)`,
		r.ID, r.GameSessionID, r.RequestedByPlayerID, r.Status, encodeTime(r.CreatedAt))
	return err
}

func (q *Queries) GetPendingFinishRequest(ctx context.Context, gameSessionID string) (*domain.FinishRequest, error) {
	var r domain.FinishRequest
	var createdAt string
	var decidedAt sql.NullString
	var decidedByPlayerID sql.NullString
	err := q.db.QueryRowContext(ctx, `SELECT id, game_session_id, requested_by_player_id, status, created_at, decided_at, decided_by_player_id FROM finish_requests WHERE game_session_id = ? AND status = 'pending' LIMIT 1`, gameSessionID).
		Scan(&r.ID, &r.GameSessionID, &r.RequestedByPlayerID, &r.Status, &createdAt, &decidedAt, &decidedByPlayerID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	r.CreatedAt, err = decodeTime(createdAt)
	if err != nil {
		return nil, err
	}
	r.DecidedAt, _ = nullTimePtr(decidedAt)
	if decidedByPlayerID.Valid {
		r.DecidedByPlayerID = &decidedByPlayerID.String
	}
	return &r, nil
}

func (q *Queries) UpdateFinishRequestStatus(ctx context.Context, id string, status domain.FinishRequestStatus, decidedAt *time.Time, decidedByPlayerID *string) error {
	_, err := q.db.ExecContext(ctx, `UPDATE finish_requests SET status = ?, decided_at = ?, decided_by_player_id = ? WHERE id = ?`,
		status, nullTimeToSQL(decidedAt), decidedByPlayerID, id)
	return err
}

func (q *Queries) ListFinishRequests(ctx context.Context, gameSessionID string) ([]domain.FinishRequest, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT id, game_session_id, requested_by_player_id, status, created_at, decided_at, decided_by_player_id FROM finish_requests WHERE game_session_id = ? ORDER BY created_at DESC`, gameSessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var requests []domain.FinishRequest
	for rows.Next() {
		var r domain.FinishRequest
		var createdAt string
		var decidedAt sql.NullString
		var decidedByPlayerID sql.NullString
		if err := rows.Scan(&r.ID, &r.GameSessionID, &r.RequestedByPlayerID, &r.Status, &createdAt, &decidedAt, &decidedByPlayerID); err != nil {
			return nil, err
		}
		r.CreatedAt, err = decodeTime(createdAt)
		if err != nil {
			return nil, err
		}
		r.DecidedAt, _ = nullTimePtr(decidedAt)
		if decidedByPlayerID.Valid {
			r.DecidedByPlayerID = &decidedByPlayerID.String
		}
		requests = append(requests, r)
	}
	return requests, rows.Err()
}

func (q *Queries) GetFinishRequest(ctx context.Context, id string) (domain.FinishRequest, error) {
	var r domain.FinishRequest
	var createdAt string
	var decidedAt sql.NullString
	var decidedByPlayerID sql.NullString
	err := q.db.QueryRowContext(ctx, `SELECT id, game_session_id, requested_by_player_id, status, created_at, decided_at, decided_by_player_id FROM finish_requests WHERE id = ?`, id).
		Scan(&r.ID, &r.GameSessionID, &r.RequestedByPlayerID, &r.Status, &createdAt, &decidedAt, &decidedByPlayerID)
	if err == sql.ErrNoRows {
		return r, domain.ErrNotFound
	}
	if err != nil {
		return r, err
	}
	r.CreatedAt, err = decodeTime(createdAt)
	if err != nil {
		return r, err
	}
	r.DecidedAt, _ = nullTimePtr(decidedAt)
	if decidedByPlayerID.Valid {
		r.DecidedByPlayerID = &decidedByPlayerID.String
	}
	return r, nil
}

// ListSettledGamesForUser returns finished, non-voided game sessions where the user has
// a historical player record, ordered by settled_at DESC. Cursor-based pagination
// is supported via beforeSettledAt.
func (q *Queries) ListSettledGamesForUser(ctx context.Context, userID string, beforeSettledAt *time.Time, limit int) ([]domain.GameSession, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var cursor interface{}
	if beforeSettledAt != nil {
		cursor = encodeTime(*beforeSettledAt)
	}
	rows, err := q.db.QueryContext(ctx,
		`SELECT gs.id, gs.name, gs.invite_code, gs.owner_user_id, gs.status, gs.max_participants,
		        gs.round_count, gs.version, gs.public_share_token, gs.last_scored_at,
		        gs.settled_at, gs.voided_at, gs.created_at, gs.updated_at
		 FROM game_sessions gs
		 JOIN players p ON p.game_session_id = gs.id
		 WHERE p.user_id = ?
		   AND gs.status = 'finished'
		   AND gs.settled_at IS NOT NULL
		   AND (? IS NULL OR gs.settled_at < ?)
		 ORDER BY gs.settled_at DESC
		 LIMIT ?`, userID, cursor, cursor, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var sessions []domain.GameSession
	for rows.Next() {
		g, err := scanGame(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, g)
	}
	return sessions, rows.Err()
}

// GetGameSessionByPublicShareToken returns a finished game session with the given public share token.
func (q *Queries) GetGameSessionByPublicShareToken(ctx context.Context, shareToken string) (domain.GameSession, error) {
	row := q.db.QueryRowContext(ctx,
		`SELECT id, name, invite_code, owner_user_id, status, max_participants,
		        round_count, version, public_share_token, last_scored_at,
		        settled_at, voided_at, created_at, updated_at
		 FROM game_sessions
		 WHERE public_share_token = ? AND status = 'finished'`, shareToken)
	g, err := scanGame(row)
	if errors.Is(err, domain.ErrNotFound) {
		return g, domain.ErrPublicShareUnavailable
	}
	return g, err
}

// ListHistoricalPlayersForUser returns all historical player records (active and inactive)
// for a given game session where the user has a historical player record.
func (q *Queries) ListHistoricalPlayersForUser(ctx context.Context, gameSessionID, userID string) ([]domain.Player, error) {
	rows, err := q.db.QueryContext(ctx,
		`SELECT p.id, p.game_session_id, p.user_id, p.display_name, p.total_score, p.active,
		        p.joined_order, p.left_at, p.created_at, p.updated_at
		 FROM players p
		 WHERE p.game_session_id = ?
		   AND EXISTS (SELECT 1 FROM players WHERE game_session_id = ? AND user_id = ?)
		 ORDER BY p.total_score DESC, p.joined_order ASC`, gameSessionID, gameSessionID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPlayers(rows)
}

func (q *Queries) FinishGameSessionWithSettleAndToken(ctx context.Context, gameSessionID string, shareToken string, now time.Time) (domain.GameSession, error) {
	_, err := q.db.ExecContext(ctx, `UPDATE game_sessions SET status = ?, settled_at = ?, public_share_token = ?, version = version + 1, updated_at = ? WHERE id = ?`,
		domain.GameSessionStatusFinished, encodeTime(now), shareToken, encodeTime(now), gameSessionID)
	if err != nil {
		return domain.GameSession{}, err
	}
	return q.GetGameSession(ctx, gameSessionID)
}
