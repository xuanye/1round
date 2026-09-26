package sqlite

import (
	"context"
	"time"
)

type PerformanceRow struct {
	GameID, GameName, UserID, PlayerID, DisplayName, AvatarURL string
	SettledAt                                                  time.Time
	Score                                                      int
}

// Only participants of the caller's settled games are returned.
func (q *Queries) ListPerformance(ctx context.Context, userID string, start, end time.Time) ([]PerformanceRow, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT g.id, g.name, g.settled_at,
 COALESCE(p.user_id, ''), p.id, p.display_name, COALESCE(u.avatar_url, ''), p.total_score
 FROM game_sessions g JOIN players p ON p.game_session_id = g.id
 LEFT JOIN users u ON u.id = p.user_id
 WHERE g.status = 'finished' AND g.voided_at IS NULL
 AND julianday(g.settled_at) >= julianday(?) AND julianday(g.settled_at) < julianday(?)
 AND EXISTS (SELECT 1 FROM players me WHERE me.game_session_id = g.id AND me.user_id = ?)
 ORDER BY g.settled_at, g.id, p.joined_order, p.id`, encodeTime(start), encodeTime(end), userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]PerformanceRow, 0)
	for rows.Next() {
		var row PerformanceRow
		var settled string
		if err := rows.Scan(&row.GameID, &row.GameName, &settled, &row.UserID, &row.PlayerID, &row.DisplayName, &row.AvatarURL, &row.Score); err != nil {
			return nil, err
		}
		row.SettledAt, err = decodeTime(settled)
		if err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}
