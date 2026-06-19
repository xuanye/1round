package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/xuanye/one-round/apps/server/internal/domain"
)

// InsertScoreTransferRaw inserts a score transfer and its receivers.
// It must be called within an existing transaction (e.g. via Store.InTx).
func (q *Queries) InsertScoreTransferRaw(ctx context.Context, t domain.ScoreTransfer) error {
	_, err := q.db.ExecContext(ctx, `INSERT INTO score_transfers (id, game_session_id, sequence_no, from_player_id, created_by_user_id, idempotency_key, amount, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.GameSessionID, t.SequenceNo, t.FromPlayerID, t.CreatedByUserID, t.IdempotencyKey, t.Amount, encodeTime(t.CreatedAt))
	if err != nil {
		return err
	}
	for _, r := range t.Receivers {
		_, err := q.db.ExecContext(ctx, `INSERT INTO score_transfer_receivers (id, score_transfer_id, player_id, receiver_order) VALUES (?, ?, ?, ?)`,
			r.ID, r.ScoreTransferID, r.PlayerID, r.ReceiverOrder)
		if err != nil {
			return err
		}
	}
	return nil
}

func (q *Queries) GetScoreTransfer(ctx context.Context, id string) (domain.ScoreTransfer, error) {
	var t domain.ScoreTransfer
	var createdAt string
	err := q.db.QueryRowContext(ctx, `SELECT id, game_session_id, sequence_no, from_player_id, created_by_user_id, idempotency_key, amount, created_at FROM score_transfers WHERE id = ?`, id).
		Scan(&t.ID, &t.GameSessionID, &t.SequenceNo, &t.FromPlayerID, &t.CreatedByUserID, &t.IdempotencyKey, &t.Amount, &createdAt)
	if err == sql.ErrNoRows {
		return t, domain.ErrNotFound
	}
	if err != nil {
		return t, err
	}
	t.CreatedAt, err = decodeTime(createdAt)
	if err != nil {
		return t, err
	}
	receivers, err := q.listScoreTransferReceivers(ctx, id)
	if err != nil {
		return t, err
	}
	t.Receivers = receivers
	return t, nil
}

func (q *Queries) listScoreTransferReceivers(ctx context.Context, scoreTransferID string) ([]domain.ScoreTransferReceiver, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT id, score_transfer_id, player_id, receiver_order FROM score_transfer_receivers WHERE score_transfer_id = ? ORDER BY receiver_order ASC`, scoreTransferID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var receivers []domain.ScoreTransferReceiver
	for rows.Next() {
		var r domain.ScoreTransferReceiver
		if err := rows.Scan(&r.ID, &r.ScoreTransferID, &r.PlayerID, &r.ReceiverOrder); err != nil {
			return nil, err
		}
		receivers = append(receivers, r)
	}
	return receivers, rows.Err()
}

func (q *Queries) NextScoreTransferSequence(ctx context.Context, gameSessionID string) (int, error) {
	var max sql.NullInt64
	err := q.db.QueryRowContext(ctx, `SELECT MAX(sequence_no) FROM score_transfers WHERE game_session_id = ?`, gameSessionID).Scan(&max)
	if err != nil {
		return 0, err
	}
	if !max.Valid {
		return 1, nil
	}
	return int(max.Int64) + 1, nil
}

func (q *Queries) ListScoreTransfers(ctx context.Context, gameSessionID string, limit int) ([]domain.ScoreTransfer, error) {
	return q.ListScoreTransfersPaginated(ctx, gameSessionID, nil, limit)
}

// ListScoreTransfersPaginated returns score transfers for a game, optionally
// filtered by beforeSequenceNo for cursor-based pagination, ordered by
// sequence_no DESC.
func (q *Queries) ListScoreTransfersPaginated(ctx context.Context, gameSessionID string, beforeSequenceNo *int, limit int) ([]domain.ScoreTransfer, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var seqNo interface{}
	if beforeSequenceNo != nil {
		seqNo = *beforeSequenceNo
	}
	rows, err := q.db.QueryContext(ctx,
		`SELECT id, game_session_id, sequence_no, from_player_id, created_by_user_id, idempotency_key, amount, created_at
		 FROM score_transfers
		 WHERE game_session_id = ?
		   AND (? IS NULL OR sequence_no < ?)
		 ORDER BY sequence_no DESC
		 LIMIT ?`, gameSessionID, seqNo, seqNo, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var transfers []domain.ScoreTransfer
	for rows.Next() {
		var t domain.ScoreTransfer
		var createdAt string
		if err := rows.Scan(&t.ID, &t.GameSessionID, &t.SequenceNo, &t.FromPlayerID, &t.CreatedByUserID, &t.IdempotencyKey, &t.Amount, &createdAt); err != nil {
			return nil, err
		}
		t.CreatedAt, err = decodeTime(createdAt)
		if err != nil {
			return nil, err
		}
		receivers, err := q.listScoreTransferReceivers(ctx, t.ID)
		if err != nil {
			return nil, err
		}
		t.Receivers = receivers
		transfers = append(transfers, t)
	}
	return transfers, rows.Err()
}

func (q *Queries) GetScoreTransferByIdempotencyKey(ctx context.Context, gameSessionID, userID, key string) (domain.ScoreTransfer, error) {
	var t domain.ScoreTransfer
	var createdAt string
	err := q.db.QueryRowContext(ctx,
		`SELECT id, game_session_id, sequence_no, from_player_id, created_by_user_id, idempotency_key, amount, created_at
		 FROM score_transfers
		 WHERE game_session_id = ? AND created_by_user_id = ? AND idempotency_key = ?`,
		gameSessionID, userID, key).
		Scan(&t.ID, &t.GameSessionID, &t.SequenceNo, &t.FromPlayerID, &t.CreatedByUserID, &t.IdempotencyKey, &t.Amount, &createdAt)
	if err == sql.ErrNoRows {
		return t, domain.ErrNotFound
	}
	if err != nil {
		return t, err
	}
	t.CreatedAt, err = decodeTime(createdAt)
	if err != nil {
		return t, err
	}
	receivers, err := q.listScoreTransferReceivers(ctx, t.ID)
	if err != nil {
		return t, err
	}
	t.Receivers = receivers
	return t, nil
}

// DebitPlayerScore decreases a player's total_score. Must be called in a transaction.
func (q *Queries) DebitPlayerScore(ctx context.Context, gameSessionID, playerID string, amount int, now time.Time) error {
	res, err := q.db.ExecContext(ctx, `UPDATE players SET total_score = total_score - ?, updated_at = ? WHERE id = ? AND game_session_id = ? AND active = 1`,
		amount, encodeTime(now), playerID, gameSessionID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrParticipantInactive
	}
	return nil
}

// CreditPlayerScore increases a player's total_score. Must be called in a transaction.
func (q *Queries) CreditPlayerScore(ctx context.Context, gameSessionID, playerID string, amount int, now time.Time) error {
	res, err := q.db.ExecContext(ctx, `UPDATE players SET total_score = total_score + ?, updated_at = ? WHERE id = ? AND game_session_id = ? AND active = 1`,
		amount, encodeTime(now), playerID, gameSessionID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrParticipantInactive
	}
	return nil
}

// IncrementGameSessionForTransfer bumps round_count, version, updated_at, and last_scored_at.
// Must be called in a transaction.
func (q *Queries) IncrementGameSessionForTransfer(ctx context.Context, gameSessionID string, nextVersion int64, now time.Time) error {
	res, err := q.db.ExecContext(ctx, `UPDATE game_sessions SET round_count = round_count + 1, version = ?, updated_at = ?, last_scored_at = ? WHERE id = ? AND status = 'active'`,
		nextVersion, encodeTime(now), encodeTime(now), gameSessionID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrGameSessionFinished
	}
	return nil
}
