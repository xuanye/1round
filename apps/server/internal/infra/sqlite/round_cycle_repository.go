package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/xuanye/one-round/apps/server/internal/domain"
)

func (q *Queries) GetActiveRoundCycle(ctx context.Context, gameSessionID string) (domain.RoundCycle, error) {
	var rc domain.RoundCycle
	var createdAt string
	var completedAt sql.NullString

	err := q.db.QueryRowContext(ctx,
		`SELECT id, game_session_id, round_no, status, created_at, completed_at
		 FROM round_cycles
		 WHERE game_session_id = ? AND status = 'active'`, gameSessionID).
		Scan(&rc.ID, &rc.GameSessionID, &rc.RoundNo, &rc.Status, &createdAt, &completedAt)
	if err == sql.ErrNoRows {
		return rc, domain.ErrNotFound
	}
	if err != nil {
		return rc, err
	}

	rc.CreatedAt, err = decodeTime(createdAt)
	if err != nil {
		return rc, err
	}
	if completedAt.Valid {
		t, err := decodeTime(completedAt.String)
		if err != nil {
			return rc, err
		}
		rc.CompletedAt = &t
	}
	return rc, nil
}

func (q *Queries) GetLatestRoundCycle(ctx context.Context, gameSessionID string) (domain.RoundCycle, error) {
	var rc domain.RoundCycle
	var createdAt string
	var completedAt sql.NullString

	err := q.db.QueryRowContext(ctx,
		`SELECT id, game_session_id, round_no, status, created_at, completed_at
		 FROM round_cycles
		 WHERE game_session_id = ?
		 ORDER BY round_no DESC
		 LIMIT 1`, gameSessionID).
		Scan(&rc.ID, &rc.GameSessionID, &rc.RoundNo, &rc.Status, &createdAt, &completedAt)
	if err == sql.ErrNoRows {
		return rc, domain.ErrNotFound
	}
	if err != nil {
		return rc, err
	}

	rc.CreatedAt, err = decodeTime(createdAt)
	if err != nil {
		return rc, err
	}
	if completedAt.Valid {
		t, err := decodeTime(completedAt.String)
		if err != nil {
			return rc, err
		}
		rc.CompletedAt = &t
	}
	return rc, nil
}

func (q *Queries) CreateRoundCycle(ctx context.Context, rc domain.RoundCycle) error {
	var completedAt interface{}
	if rc.CompletedAt != nil {
		completedAt = encodeTime(*rc.CompletedAt)
	}
	_, err := q.db.ExecContext(ctx,
		`INSERT INTO round_cycles (id, game_session_id, round_no, status, created_at, completed_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		rc.ID, rc.GameSessionID, rc.RoundNo, string(rc.Status), encodeTime(rc.CreatedAt), completedAt)
	return err
}

func (q *Queries) ListRoundParticipationStatuses(ctx context.Context, roundCycleID string) ([]domain.RoundParticipationStatus, error) {
	rows, err := q.db.QueryContext(ctx,
		`SELECT id, round_cycle_id, player_id, status, satisfied_by_transfer_id, updated_at
		 FROM round_participation_statuses
		 WHERE round_cycle_id = ?`, roundCycleID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var statuses []domain.RoundParticipationStatus
	for rows.Next() {
		var s domain.RoundParticipationStatus
		var satisfiedBy sql.NullString
		var updatedAt string
		if err := rows.Scan(&s.ID, &s.RoundCycleID, &s.PlayerID, &s.Status, &satisfiedBy, &updatedAt); err != nil {
			return nil, err
		}
		if satisfiedBy.Valid {
			s.SatisfiedByTransfer = &satisfiedBy.String
		}
		s.UpdatedAt, err = decodeTime(updatedAt)
		if err != nil {
			return nil, err
		}
		statuses = append(statuses, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return statuses, nil
}

func (q *Queries) UpsertRoundParticipationStatus(ctx context.Context, s domain.RoundParticipationStatus) error {
	var satisfiedBy interface{}
	if s.SatisfiedByTransfer != nil {
		satisfiedBy = *s.SatisfiedByTransfer
	}
	_, err := q.db.ExecContext(ctx,
		`INSERT INTO round_participation_statuses (id, round_cycle_id, player_id, status, satisfied_by_transfer_id, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(round_cycle_id, player_id) DO UPDATE SET
			status = excluded.status,
			satisfied_by_transfer_id = excluded.satisfied_by_transfer_id,
			updated_at = excluded.updated_at`,
		s.ID, s.RoundCycleID, s.PlayerID, string(s.Status), satisfiedBy, encodeTime(s.UpdatedAt))
	return err
}

func (q *Queries) SetRoundParticipationStatus(ctx context.Context, roundCycleID, playerID string, status domain.ParticipationStatus, now time.Time) error {
	_, err := q.db.ExecContext(ctx,
		`UPDATE round_participation_statuses
		 SET status = ?, satisfied_by_transfer_id = NULL, updated_at = ?
		 WHERE round_cycle_id = ? AND player_id = ?`,
		string(status), encodeTime(now), roundCycleID, playerID)
	return err
}

func (q *Queries) CompleteRoundCycle(ctx context.Context, roundCycleID string, completedAt time.Time) error {
	_, err := q.db.ExecContext(ctx,
		`UPDATE round_cycles
		 SET status = 'complete', completed_at = ?
		 WHERE id = ?`, encodeTime(completedAt), roundCycleID)
	return err
}

func (q *Queries) ResetRoundParticipationAfterReversal(ctx context.Context, roundCycleID, transferID string, now time.Time) error {
	_, err := q.db.ExecContext(ctx,
		`UPDATE round_participation_statuses
		 SET status = 'pending', satisfied_by_transfer_id = NULL, updated_at = ?
		 WHERE round_cycle_id = ? AND satisfied_by_transfer_id = ?`,
		encodeTime(now), roundCycleID, transferID)
	return err
}

func (q *Queries) ReopenRoundCycle(ctx context.Context, roundCycleID string) error {
	_, err := q.db.ExecContext(ctx,
		`UPDATE round_cycles
		 SET status = 'active', completed_at = NULL
		 WHERE id = ?`, roundCycleID)
	return err
}
