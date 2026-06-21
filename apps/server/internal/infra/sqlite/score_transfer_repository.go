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
	var roundCycleID interface{}
	if t.RoundCycleID != "" {
		roundCycleID = t.RoundCycleID
	}
	var reversalOf interface{}
	if t.ReversalOfTransferID != nil {
		reversalOf = *t.ReversalOfTransferID
	}
	var reversedAt interface{}
	if t.ReversedAt != nil {
		reversedAt = encodeTime(*t.ReversedAt)
	}

	_, err := q.db.ExecContext(ctx,
		`INSERT INTO score_transfers (id, game_session_id, round_cycle_id, sequence_no, from_player_id, created_by_user_id, idempotency_key, amount, transfer_kind, reversal_of_transfer_id, reversed_at, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.GameSessionID, roundCycleID, t.SequenceNo, t.FromPlayerID, t.CreatedByUserID, t.IdempotencyKey, t.Amount, string(t.Kind), reversalOf, reversedAt, encodeTime(t.CreatedAt))
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
	var roundCycleID sql.NullString
	var kind string
	var reversalOf sql.NullString
	var reversedAt sql.NullString

	err := q.db.QueryRowContext(ctx,
		`SELECT id, game_session_id, round_cycle_id, sequence_no, from_player_id, created_by_user_id, idempotency_key, amount, transfer_kind, reversal_of_transfer_id, reversed_at, created_at
		 FROM score_transfers WHERE id = ?`, id).
		Scan(&t.ID, &t.GameSessionID, &roundCycleID, &t.SequenceNo, &t.FromPlayerID, &t.CreatedByUserID, &t.IdempotencyKey, &t.Amount, &kind, &reversalOf, &reversedAt, &createdAt)
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
	if roundCycleID.Valid {
		t.RoundCycleID = roundCycleID.String
	}
	t.Kind = domain.ScoreTransferKind(kind)
	if reversalOf.Valid {
		t.ReversalOfTransferID = &reversalOf.String
	}
	if reversedAt.Valid {
		rat, err := decodeTime(reversedAt.String)
		if err != nil {
			return t, err
		}
		t.ReversedAt = &rat
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
	defer func() { _ = rows.Close() }()
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
		`SELECT id, game_session_id, round_cycle_id, sequence_no, from_player_id, created_by_user_id, idempotency_key, amount, transfer_kind, reversal_of_transfer_id, reversed_at, created_at
		 FROM score_transfers
		 WHERE game_session_id = ?
		   AND (? IS NULL OR sequence_no < ?)
		 ORDER BY sequence_no DESC
		 LIMIT ?`, gameSessionID, seqNo, seqNo, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var transfers []domain.ScoreTransfer
	for rows.Next() {
		var t domain.ScoreTransfer
		var roundCycleID sql.NullString
		var kind string
		var reversalOf sql.NullString
		var reversedAt sql.NullString
		var createdAt string
		if err := rows.Scan(&t.ID, &t.GameSessionID, &roundCycleID, &t.SequenceNo, &t.FromPlayerID, &t.CreatedByUserID, &t.IdempotencyKey, &t.Amount, &kind, &reversalOf, &reversedAt, &createdAt); err != nil {
			return nil, err
		}
		t.CreatedAt, err = decodeTime(createdAt)
		if err != nil {
			return nil, err
		}
		if roundCycleID.Valid {
			t.RoundCycleID = roundCycleID.String
		}
		t.Kind = domain.ScoreTransferKind(kind)
		if reversalOf.Valid {
			t.ReversalOfTransferID = &reversalOf.String
		}
		if reversedAt.Valid {
			rat, err := decodeTime(reversedAt.String)
			if err != nil {
				return nil, err
			}
			t.ReversedAt = &rat
		}
		transfers = append(transfers, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	_ = rows.Close()

	for i := range transfers {
		receivers, err := q.listScoreTransferReceivers(ctx, transfers[i].ID)
		if err != nil {
			return nil, err
		}
		transfers[i].Receivers = receivers
	}
	return transfers, nil
}

func (q *Queries) GetScoreTransferByIdempotencyKey(ctx context.Context, gameSessionID, userID, key string) (domain.ScoreTransfer, error) {
	var t domain.ScoreTransfer
	var createdAt string
	var roundCycleID sql.NullString
	var kind string
	var reversalOf sql.NullString
	var reversedAt sql.NullString

	err := q.db.QueryRowContext(ctx,
		`SELECT id, game_session_id, round_cycle_id, sequence_no, from_player_id, created_by_user_id, idempotency_key, amount, transfer_kind, reversal_of_transfer_id, reversed_at, created_at
		 FROM score_transfers
		 WHERE game_session_id = ? AND created_by_user_id = ? AND idempotency_key = ?`,
		gameSessionID, userID, key).
		Scan(&t.ID, &t.GameSessionID, &roundCycleID, &t.SequenceNo, &t.FromPlayerID, &t.CreatedByUserID, &t.IdempotencyKey, &t.Amount, &kind, &reversalOf, &reversedAt, &createdAt)
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
	if roundCycleID.Valid {
		t.RoundCycleID = roundCycleID.String
	}
	t.Kind = domain.ScoreTransferKind(kind)
	if reversalOf.Valid {
		t.ReversalOfTransferID = &reversalOf.String
	}
	if reversedAt.Valid {
		rat, err := decodeTime(reversedAt.String)
		if err != nil {
			return t, err
		}
		t.ReversedAt = &rat
	}

	receivers, err := q.listScoreTransferReceivers(ctx, t.ID)
	if err != nil {
		return t, err
	}
	t.Receivers = receivers
	return t, nil
}

func (q *Queries) GetScoreTransferForUpdate(ctx context.Context, gameSessionID, transferID string) (domain.ScoreTransfer, error) {
	var t domain.ScoreTransfer
	var createdAt string
	var roundCycleID sql.NullString
	var kind string
	var reversalOf sql.NullString
	var reversedAt sql.NullString

	err := q.db.QueryRowContext(ctx,
		`SELECT id, game_session_id, round_cycle_id, sequence_no, from_player_id, created_by_user_id, idempotency_key, amount, transfer_kind, reversal_of_transfer_id, reversed_at, created_at
		 FROM score_transfers
		 WHERE game_session_id = ? AND id = ?`,
		gameSessionID, transferID).
		Scan(&t.ID, &t.GameSessionID, &roundCycleID, &t.SequenceNo, &t.FromPlayerID, &t.CreatedByUserID, &t.IdempotencyKey, &t.Amount, &kind, &reversalOf, &reversedAt, &createdAt)
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
	if roundCycleID.Valid {
		t.RoundCycleID = roundCycleID.String
	}
	t.Kind = domain.ScoreTransferKind(kind)
	if reversalOf.Valid {
		t.ReversalOfTransferID = &reversalOf.String
	}
	if reversedAt.Valid {
		rat, err := decodeTime(reversedAt.String)
		if err != nil {
			return t, err
		}
		t.ReversedAt = &rat
	}

	receivers, err := q.listScoreTransferReceivers(ctx, t.ID)
	if err != nil {
		return t, err
	}
	t.Receivers = receivers
	return t, nil
}

func (q *Queries) MarkScoreTransferReversed(ctx context.Context, transferID string, reversedAt time.Time) error {
	_, err := q.db.ExecContext(ctx, `UPDATE score_transfers SET reversed_at = ? WHERE id = ?`, encodeTime(reversedAt), transferID)
	return err
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
// Uses atomic version = version + 1 to prevent concurrent regressions.
// Returns the new version. Must be called in a transaction.
func (q *Queries) IncrementGameSessionForTransfer(ctx context.Context, gameSessionID string, now time.Time) (int64, error) {
	res, err := q.db.ExecContext(ctx, `UPDATE game_sessions SET round_count = round_count + 1, version = version + 1, updated_at = ?, last_scored_at = ? WHERE id = ? AND status = 'active'`,
		encodeTime(now), encodeTime(now), gameSessionID)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return 0, domain.ErrGameSessionFinished
	}
	// Read back the new version
	var newVersion int64
	err = q.db.QueryRowContext(ctx, `SELECT version FROM game_sessions WHERE id = ?`, gameSessionID).Scan(&newVersion)
	if err != nil {
		return 0, err
	}
	return newVersion, nil
}
