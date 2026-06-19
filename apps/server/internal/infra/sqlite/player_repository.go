package sqlite

import (
	"context"
	"database/sql"

	"github.com/redhu/one-round/apps/server/internal/domain"
)

func (q *Queries) CreatePlayer(ctx context.Context, p domain.Player) error {
	_, err := q.db.ExecContext(ctx, `INSERT INTO players (id, game_session_id, user_id, display_name, total_score, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.GameSessionID, p.UserID, p.DisplayName, p.TotalScore, encodeTime(p.CreatedAt), encodeTime(p.UpdatedAt))
	return err
}

func (q *Queries) ListPlayers(ctx context.Context, gameSessionID string) ([]domain.Player, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT id, game_session_id, user_id, display_name, total_score, created_at, updated_at FROM players WHERE game_session_id = ? ORDER BY created_at ASC`, gameSessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var players []domain.Player
	for rows.Next() {
		var p domain.Player
		var createdAt, updatedAt string
		if err := rows.Scan(&p.ID, &p.GameSessionID, &p.UserID, &p.DisplayName, &p.TotalScore, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		p.CreatedAt, err = decodeTime(createdAt)
		if err != nil {
			return nil, err
		}
		p.UpdatedAt, err = decodeTime(updatedAt)
		if err != nil {
			return nil, err
		}
		players = append(players, p)
	}
	return players, rows.Err()
}

func (q *Queries) ListRanking(ctx context.Context, gameSessionID string) ([]domain.Player, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT id, game_session_id, user_id, display_name, total_score, created_at, updated_at FROM players WHERE game_session_id = ? ORDER BY total_score DESC, display_name ASC`, gameSessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var players []domain.Player
	for rows.Next() {
		var p domain.Player
		var createdAt, updatedAt string
		if err := rows.Scan(&p.ID, &p.GameSessionID, &p.UserID, &p.DisplayName, &p.TotalScore, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		p.CreatedAt, err = decodeTime(createdAt)
		if err != nil {
			return nil, err
		}
		p.UpdatedAt, err = decodeTime(updatedAt)
		if err != nil {
			return nil, err
		}
		players = append(players, p)
	}
	return players, rows.Err()
}

func (q *Queries) UpdatePlayer(ctx context.Context, gameSessionID, playerID, displayName string) (domain.Player, error) {
	_, err := q.db.ExecContext(ctx, `UPDATE players SET display_name = ?, updated_at = datetime('now') WHERE id = ? AND game_session_id = ?`, displayName, playerID, gameSessionID)
	if err != nil {
		return domain.Player{}, err
	}
	return q.GetPlayer(ctx, gameSessionID, playerID)
}

func (q *Queries) DeletePlayer(ctx context.Context, gameSessionID, playerID string) error {
	var scoreID string
	err := q.db.QueryRowContext(ctx, `SELECT id FROM round_scores WHERE player_id = ? LIMIT 1`, playerID).Scan(&scoreID)
	if err == nil {
		return domain.ErrConflict
	}
	if err != sql.ErrNoRows {
		return err
	}
	res, err := q.db.ExecContext(ctx, `DELETE FROM players WHERE id = ? AND game_session_id = ?`, playerID, gameSessionID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (q *Queries) GetPlayer(ctx context.Context, gameSessionID, playerID string) (domain.Player, error) {
	var p domain.Player
	var createdAt, updatedAt string
	err := q.db.QueryRowContext(ctx, `SELECT id, game_session_id, user_id, display_name, total_score, created_at, updated_at FROM players WHERE id = ? AND game_session_id = ?`, playerID, gameSessionID).
		Scan(&p.ID, &p.GameSessionID, &p.UserID, &p.DisplayName, &p.TotalScore, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return p, domain.ErrNotFound
	}
	if err == nil {
		p.CreatedAt, err = decodeTime(createdAt)
		if err != nil {
			return p, err
		}
		p.UpdatedAt, err = decodeTime(updatedAt)
	}
	return p, err
}
