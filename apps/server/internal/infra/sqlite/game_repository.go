package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/xuanye/one-round/apps/server/internal/domain"
)

func (q *Queries) CreateGameSession(ctx context.Context, session domain.GameSession, ownerMember domain.GameMember) error {
	_, err := q.db.ExecContext(ctx, `INSERT INTO game_sessions (id, name, invite_code, owner_user_id, status, zero_sum_required, round_count, version, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		session.ID, session.Name, session.InviteCode, session.OwnerUserID, session.Status, boolInt(session.ZeroSumRequired), session.RoundCount, session.Version, encodeTime(session.CreatedAt), encodeTime(session.UpdatedAt))
	if err != nil {
		return err
	}
	_, err = q.db.ExecContext(ctx, `INSERT INTO game_members (id, game_session_id, user_id, role, joined_at) VALUES (?, ?, ?, ?, ?)`,
		ownerMember.ID, ownerMember.GameSessionID, ownerMember.UserID, ownerMember.Role, encodeTime(ownerMember.JoinedAt))
	return err
}

func (q *Queries) AddGameMember(ctx context.Context, m domain.GameMember) error {
	_, err := q.db.ExecContext(ctx, `INSERT OR IGNORE INTO game_members (id, game_session_id, user_id, role, joined_at) VALUES (?, ?, ?, ?, ?)`, m.ID, m.GameSessionID, m.UserID, m.Role, encodeTime(m.JoinedAt))
	return err
}

func (q *Queries) GetGameSession(ctx context.Context, id string) (domain.GameSession, error) {
	row := q.db.QueryRowContext(ctx, `SELECT id, name, invite_code, owner_user_id, status, zero_sum_required, round_count, version, created_at, updated_at FROM game_sessions WHERE id = ?`, id)
	return scanGame(row)
}

func (q *Queries) GetGameSessionByInviteCode(ctx context.Context, inviteCode string) (domain.GameSession, error) {
	row := q.db.QueryRowContext(ctx, `SELECT id, name, invite_code, owner_user_id, status, zero_sum_required, round_count, version, created_at, updated_at FROM game_sessions WHERE invite_code = ?`, inviteCode)
	return scanGame(row)
}

func (q *Queries) IsGameMember(ctx context.Context, gameSessionID, userID string) (bool, error) {
	var id string
	err := q.db.QueryRowContext(ctx, `SELECT id FROM game_members WHERE game_session_id = ? AND user_id = ?`, gameSessionID, userID).Scan(&id)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

func (q *Queries) FinishGameSession(ctx context.Context, gameSessionID string, now time.Time) (domain.GameSession, error) {
	_, err := q.db.ExecContext(ctx, `UPDATE game_sessions SET status = ?, version = version + 1, updated_at = ? WHERE id = ?`, domain.GameSessionStatusFinished, encodeTime(now), gameSessionID)
	if err != nil {
		return domain.GameSession{}, err
	}
	return q.GetGameSession(ctx, gameSessionID)
}

func scanGame(row interface{ Scan(...any) error }) (domain.GameSession, error) {
	var g domain.GameSession
	var zero int
	var createdAt, updatedAt string
	err := row.Scan(&g.ID, &g.Name, &g.InviteCode, &g.OwnerUserID, &g.Status, &zero, &g.RoundCount, &g.Version, &createdAt, &updatedAt)
	g.ZeroSumRequired = zero == 1
	if err == sql.ErrNoRows {
		return g, domain.ErrNotFound
	}
	if err == nil {
		g.CreatedAt, err = decodeTime(createdAt)
		if err != nil {
			return g, err
		}
		g.UpdatedAt, err = decodeTime(updatedAt)
	}
	return g, err
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
