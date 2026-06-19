package sqlite

import (
	"context"
	"database/sql"
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
