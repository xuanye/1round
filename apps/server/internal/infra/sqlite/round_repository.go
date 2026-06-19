package sqlite

import (
	"context"
	"database/sql"

	"github.com/redhu/one-round/apps/server/internal/domain"
)

func (q *Queries) InsertRoundWithScores(ctx context.Context, r domain.Round, nextVersion int64) error {
	_, err := q.db.ExecContext(ctx, `INSERT INTO rounds (id, game_session_id, round_no, created_by_user_id, note, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		r.ID, r.GameSessionID, r.RoundNo, r.CreatedByUserID, r.Note, encodeTime(r.CreatedAt))
	if err != nil {
		return err
	}
	for _, score := range r.Scores {
		if _, err := q.db.ExecContext(ctx, `INSERT INTO round_scores (id, round_id, player_id, score) VALUES (?, ?, ?, ?)`, score.ID, score.RoundID, score.PlayerID, score.Score); err != nil {
			return err
		}
		if _, err := q.db.ExecContext(ctx, `UPDATE players SET total_score = total_score + ?, updated_at = ? WHERE id = ? AND game_session_id = ?`, score.Score, encodeTime(r.CreatedAt), score.PlayerID, r.GameSessionID); err != nil {
			return err
		}
	}
	_, err = q.db.ExecContext(ctx, `UPDATE game_sessions SET round_count = round_count + 1, version = ?, updated_at = ? WHERE id = ?`, nextVersion, encodeTime(r.CreatedAt), r.GameSessionID)
	return err
}

func (q *Queries) ListRecentRounds(ctx context.Context, gameSessionID string, limit int) ([]domain.Round, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT id, game_session_id, round_no, created_by_user_id, note, created_at FROM rounds WHERE game_session_id = ? ORDER BY round_no DESC LIMIT ?`, gameSessionID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var rounds []domain.Round
	for rows.Next() {
		var r domain.Round
		var createdAt string
		if err := rows.Scan(&r.ID, &r.GameSessionID, &r.RoundNo, &r.CreatedByUserID, &r.Note, &createdAt); err != nil {
			return nil, err
		}
		r.CreatedAt, err = decodeTime(createdAt)
		if err != nil {
			return nil, err
		}
		rounds = append(rounds, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	for i := range rounds {
		scores, err := q.listRoundScores(ctx, rounds[i].ID)
		if err != nil {
			return nil, err
		}
		rounds[i].Scores = scores
	}
	return rounds, nil
}

func (q *Queries) listRoundScores(ctx context.Context, roundID string) ([]domain.RoundScore, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT id, round_id, player_id, score FROM round_scores WHERE round_id = ?`, roundID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var scores []domain.RoundScore
	for rows.Next() {
		var s domain.RoundScore
		if err := rows.Scan(&s.ID, &s.RoundID, &s.PlayerID, &s.Score); err != nil {
			return nil, err
		}
		scores = append(scores, s)
	}
	return scores, rows.Err()
}

func (q *Queries) CountRoundScoresByPlayer(ctx context.Context, playerID string) (int, error) {
	var n int
	err := q.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM round_scores WHERE player_id = ?`, playerID).Scan(&n)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return n, err
}
