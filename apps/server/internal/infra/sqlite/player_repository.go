package sqlite

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/xuanye/one-round/apps/server/internal/domain"
)

func (q *Queries) CreatePlayer(ctx context.Context, p domain.Player) error {
	_, err := q.db.ExecContext(ctx, `INSERT INTO players (id, game_session_id, user_id, display_name, total_score, active, joined_order, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.GameSessionID, p.UserID, p.DisplayName, p.TotalScore, boolToInt(p.Active), p.JoinedOrder, encodeTime(p.CreatedAt), encodeTime(p.UpdatedAt))
	if err != nil {
		// Map unique constraint violations to domain errors
		if strings.Contains(err.Error(), "UNIQUE constraint failed: players.game_session_id, display_name") {
			return domain.ErrDuplicateDisplayName
		}
		if strings.Contains(err.Error(), "UNIQUE constraint failed: players.game_session_id, user_id") {
			return domain.ErrActiveGameExists
		}
		return err
	}
	return nil
}

func (q *Queries) ListPlayers(ctx context.Context, gameSessionID string) ([]domain.Player, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT id, game_session_id, user_id, display_name, total_score, active, joined_order, left_at, created_at, updated_at FROM players WHERE game_session_id = ? ORDER BY created_at ASC`, gameSessionID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanPlayers(rows)
}

func (q *Queries) ListRanking(ctx context.Context, gameSessionID string) ([]domain.Player, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT id, game_session_id, user_id, display_name, total_score, active, joined_order, left_at, created_at, updated_at FROM players WHERE game_session_id = ? AND active = 1 ORDER BY total_score DESC, joined_order ASC`, gameSessionID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanPlayers(rows)
}

func (q *Queries) ListActivePlayers(ctx context.Context, gameSessionID string) ([]domain.Player, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT id, game_session_id, user_id, display_name, total_score, active, joined_order, left_at, created_at, updated_at FROM players WHERE game_session_id = ? AND active = 1 ORDER BY joined_order ASC`, gameSessionID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanPlayers(rows)
}

func (q *Queries) ListHistoricalPlayers(ctx context.Context, gameSessionID string) ([]domain.Player, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT id, game_session_id, user_id, display_name, total_score, active, joined_order, left_at, created_at, updated_at FROM players WHERE game_session_id = ? ORDER BY joined_order ASC`, gameSessionID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanPlayers(rows)
}

func (q *Queries) CountActiveParticipants(ctx context.Context, gameSessionID string) (int, error) {
	var n int
	err := q.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM players WHERE game_session_id = ? AND active = 1`, gameSessionID).Scan(&n)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return n, err
}

func (q *Queries) NextJoinedOrder(ctx context.Context, gameSessionID string) (int, error) {
	var max sql.NullInt64
	err := q.db.QueryRowContext(ctx, `SELECT MAX(joined_order) FROM players WHERE game_session_id = ?`, gameSessionID).Scan(&max)
	if err != nil {
		return 0, err
	}
	if !max.Valid {
		return 1, nil
	}
	return int(max.Int64) + 1, nil
}

func (q *Queries) GetActivePlayerByUser(ctx context.Context, gameSessionID, userID string) (domain.Player, error) {
	return q.getPlayerByUser(ctx, gameSessionID, userID, true)
}

func (q *Queries) GetHistoricalPlayerByUser(ctx context.Context, gameSessionID, userID string) (domain.Player, error) {
	return q.getPlayerByUser(ctx, gameSessionID, userID, false)
}

func (q *Queries) getPlayerByUser(ctx context.Context, gameSessionID, userID string, activeOnly bool) (domain.Player, error) {
	var p domain.Player
	var leftAt sql.NullString
	var createdAt, updatedAt string
	query := `SELECT id, game_session_id, user_id, display_name, total_score, active, joined_order, left_at, created_at, updated_at FROM players WHERE game_session_id = ? AND user_id = ?`
	if activeOnly {
		query += ` AND active = 1`
	}
	err := q.db.QueryRowContext(ctx, query, gameSessionID, userID).
		Scan(&p.ID, &p.GameSessionID, &p.UserID, &p.DisplayName, &p.TotalScore, &p.Active, &p.JoinedOrder, &leftAt, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return p, domain.ErrNotFound
	}
	if err != nil {
		return p, err
	}
	p.LeftAt, _ = nullTimePtr(leftAt)
	p.CreatedAt, err = decodeTime(createdAt)
	if err != nil {
		return p, err
	}
	p.UpdatedAt, err = decodeTime(updatedAt)
	return p, err
}

func (q *Queries) UpdatePlayer(ctx context.Context, gameSessionID, playerID, displayName string) (domain.Player, error) {
	return q.UpdatePlayerAt(ctx, gameSessionID, playerID, displayName, time.Now().UTC())
}

func (q *Queries) UpdatePlayerAt(ctx context.Context, gameSessionID, playerID, displayName string, now time.Time) (domain.Player, error) {
	_, err := q.db.ExecContext(ctx, `UPDATE players SET display_name = ?, updated_at = ? WHERE id = ? AND game_session_id = ?`, displayName, encodeTime(now), playerID, gameSessionID)
	if err != nil {
		return domain.Player{}, err
	}
	return q.GetPlayer(ctx, gameSessionID, playerID)
}

func (q *Queries) ReactivatePlayer(ctx context.Context, playerID, gameSessionID, displayName string, totalScore int, now time.Time) error {
	_, err := q.db.ExecContext(ctx, `UPDATE players SET active = 1, display_name = ?, total_score = ?, left_at = NULL, updated_at = ? WHERE id = ? AND game_session_id = ?`,
		displayName, totalScore, encodeTime(now), playerID, gameSessionID)
	if err != nil {
		// Map unique constraint violations to domain errors
		if strings.Contains(err.Error(), "UNIQUE constraint failed: players.game_session_id, display_name") {
			return domain.ErrDuplicateDisplayName
		}
		return err
	}
	return nil
}

func (q *Queries) DeactivatePlayer(ctx context.Context, gameSessionID, userID string, now time.Time) error {
	res, err := q.db.ExecContext(ctx, `UPDATE players SET active = 0, left_at = ?, updated_at = ? WHERE game_session_id = ? AND user_id = ? AND active = 1`,
		encodeTime(now), encodeTime(now), gameSessionID, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrAlreadyDeactivated
	}
	return nil
}

func (q *Queries) GetNextOwner(ctx context.Context, gameSessionID string) (*domain.Player, error) {
	var p domain.Player
	var leftAt sql.NullString
	var createdAt, updatedAt string
	err := q.db.QueryRowContext(ctx,
		`SELECT id, game_session_id, user_id, display_name, total_score, active, joined_order, left_at, created_at, updated_at
		 FROM players WHERE game_session_id = ? AND active = 1 AND user_id IS NOT NULL
		 ORDER BY joined_order ASC LIMIT 1`, gameSessionID).
		Scan(&p.ID, &p.GameSessionID, &p.UserID, &p.DisplayName, &p.TotalScore, &p.Active, &p.JoinedOrder, &leftAt, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	p.LeftAt, _ = nullTimePtr(leftAt)
	p.CreatedAt, err = decodeTime(createdAt)
	if err != nil {
		return nil, err
	}
	p.UpdatedAt, err = decodeTime(updatedAt)
	return &p, err
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
	var leftAt sql.NullString
	var createdAt, updatedAt string
	err := q.db.QueryRowContext(ctx, `SELECT id, game_session_id, user_id, display_name, total_score, active, joined_order, left_at, created_at, updated_at FROM players WHERE id = ? AND game_session_id = ?`, playerID, gameSessionID).
		Scan(&p.ID, &p.GameSessionID, &p.UserID, &p.DisplayName, &p.TotalScore, &p.Active, &p.JoinedOrder, &leftAt, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return p, domain.ErrNotFound
	}
	if err != nil {
		return p, err
	}
	p.LeftAt, _ = nullTimePtr(leftAt)
	p.CreatedAt, err = decodeTime(createdAt)
	if err != nil {
		return p, err
	}
	p.UpdatedAt, err = decodeTime(updatedAt)
	return p, err
}

func scanPlayers(rows *sql.Rows) ([]domain.Player, error) {
	var players []domain.Player
	for rows.Next() {
		var p domain.Player
		var leftAt sql.NullString
		var createdAt, updatedAt string
		if err := rows.Scan(&p.ID, &p.GameSessionID, &p.UserID, &p.DisplayName, &p.TotalScore, &p.Active, &p.JoinedOrder, &leftAt, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		var err error
		p.LeftAt, _ = nullTimePtr(leftAt)
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

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
