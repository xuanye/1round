package sqlite

import (
	"context"
	"database/sql"

	"github.com/xuanye/one-round/apps/server/internal/domain"
)

func (q *Queries) InsertScoreTransfer(ctx context.Context, t domain.ScoreTransfer) error {
	_, err := q.db.ExecContext(ctx, `INSERT INTO score_transfers (id, game_session_id, sequence_no, from_player_id, created_by_user_id, idempotency_key, amount, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.GameSessionID, t.SequenceNo, t.FromPlayerID, t.CreatedByUserID, t.IdempotencyKey, t.Amount, encodeTime(t.CreatedAt))
	if err != nil {
		return err
	}
	for i, r := range t.Receivers {
		_, err := q.db.ExecContext(ctx, `INSERT INTO score_transfer_receivers (id, score_transfer_id, player_id, receiver_order) VALUES (?, ?, ?, ?)`,
			r.ID, r.ScoreTransferID, r.PlayerID, i+1)
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
	rows, err := q.db.QueryContext(ctx, `SELECT id, game_session_id, sequence_no, from_player_id, created_by_user_id, idempotency_key, amount, created_at FROM score_transfers WHERE game_session_id = ? ORDER BY sequence_no DESC LIMIT ?`, gameSessionID, limit)
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
