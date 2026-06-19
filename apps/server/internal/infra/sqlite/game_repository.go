package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/xuanye/one-round/apps/server/internal/domain"
)

func (q *Queries) CreateGameSession(ctx context.Context, session domain.GameSession, ownerMember domain.GameMember) error {
	_, err := q.db.ExecContext(ctx, `INSERT INTO game_sessions (id, name, invite_code, owner_user_id, status, zero_sum_required, round_count, version, max_participants, public_share_token, last_scored_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		session.ID, session.Name, session.InviteCode, session.OwnerUserID, session.Status, 0, session.ScoreTransferCnt, session.Version, nullIntToSQL(session.MaxParticipants), nullStringToSQL(session.PublicShareToken), nullTimeToSQL(session.LastScoredAt), encodeTime(session.CreatedAt), encodeTime(session.UpdatedAt))
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
	row := q.db.QueryRowContext(ctx, `SELECT id, name, invite_code, owner_user_id, status, max_participants, round_count, version, public_share_token, last_scored_at, settled_at, voided_at, created_at, updated_at FROM game_sessions WHERE id = ?`, id)
	return scanGame(row)
}

func (q *Queries) GetGameSessionByInviteCode(ctx context.Context, inviteCode string) (domain.GameSession, error) {
	row := q.db.QueryRowContext(ctx, `SELECT id, name, invite_code, owner_user_id, status, max_participants, round_count, version, public_share_token, last_scored_at, settled_at, voided_at, created_at, updated_at FROM game_sessions WHERE invite_code = ?`, inviteCode)
	return scanGame(row)
}

func (q *Queries) GetCurrentGameForUser(ctx context.Context, userID string) (*domain.GameSession, error) {
	row := q.db.QueryRowContext(ctx, `
		SELECT gs.id, gs.name, gs.invite_code, gs.owner_user_id, gs.status, gs.max_participants,
		       gs.round_count, gs.version, gs.public_share_token, gs.last_scored_at,
		       gs.settled_at, gs.voided_at, gs.created_at, gs.updated_at
		FROM game_sessions gs
		JOIN players p ON p.game_session_id = gs.id
		WHERE p.user_id = ?
		  AND p.active = 1
		  AND gs.status = 'active'
		ORDER BY gs.updated_at DESC
		LIMIT 1`, userID)
	g, err := scanGame(row)
	if err == domain.ErrNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &g, nil
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

func (q *Queries) FinishGameSessionWithSettle(ctx context.Context, gameSessionID string, now time.Time) (domain.GameSession, error) {
	_, err := q.db.ExecContext(ctx, `UPDATE game_sessions SET status = ?, settled_at = ?, version = version + 1, updated_at = ? WHERE id = ?`,
		domain.GameSessionStatusFinished, encodeTime(now), encodeTime(now), gameSessionID)
	if err != nil {
		return domain.GameSession{}, err
	}
	return q.GetGameSession(ctx, gameSessionID)
}

func (q *Queries) VoidGameSession(ctx context.Context, gameSessionID string, now time.Time) (domain.GameSession, error) {
	_, err := q.db.ExecContext(ctx, `UPDATE game_sessions SET status = ?, voided_at = ?, version = version + 1, updated_at = ? WHERE id = ?`,
		domain.GameSessionStatusVoided, encodeTime(now), encodeTime(now), gameSessionID)
	if err != nil {
		return domain.GameSession{}, err
	}
	return q.GetGameSession(ctx, gameSessionID)
}

func (q *Queries) UpdateGameSessionOwner(ctx context.Context, gameSessionID, newOwnerUserID string, now time.Time) error {
	_, err := q.db.ExecContext(ctx, `UPDATE game_sessions SET owner_user_id = ?, version = version + 1, updated_at = ? WHERE id = ?`,
		newOwnerUserID, encodeTime(now), gameSessionID)
	return err
}

func (q *Queries) IncrementGameSessionVersion(ctx context.Context, gameSessionID string, now time.Time) error {
	_, err := q.db.ExecContext(ctx, `UPDATE game_sessions SET version = version + 1, updated_at = ? WHERE id = ?`,
		encodeTime(now), gameSessionID)
	return err
}

func (q *Queries) SetPublicShareToken(ctx context.Context, gameSessionID, token string) error {
	_, err := q.db.ExecContext(ctx, `UPDATE game_sessions SET public_share_token = ? WHERE id = ?`, token, gameSessionID)
	return err
}

func (q *Queries) TransferGameMemberRole(ctx context.Context, gameSessionID, newOwnerUserID string, now time.Time) error {
	// Downgrade current owner to member
	_, err := q.db.ExecContext(ctx, `UPDATE game_members SET role = ? WHERE game_session_id = ? AND role = ?`,
		domain.GameMemberRoleMember, gameSessionID, domain.GameMemberRoleOwner)
	if err != nil {
		return err
	}
	// Upgrade new owner
	_, err = q.db.ExecContext(ctx, `UPDATE game_members SET role = ? WHERE game_session_id = ? AND user_id = ?`,
		domain.GameMemberRoleOwner, gameSessionID, newOwnerUserID)
	return err
}

func scanGame(row interface{ Scan(...any) error }) (domain.GameSession, error) {
	var g domain.GameSession
	var maxParticipants, publicShareToken, lastScoredAt, settledAt, voidedAt sql.NullString
	var roundCount int
	var version int64
	var createdAt, updatedAt string
	err := row.Scan(&g.ID, &g.Name, &g.InviteCode, &g.OwnerUserID, &g.Status, &maxParticipants,
		&roundCount, &version, &publicShareToken, &lastScoredAt, &settledAt, &voidedAt,
		&createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return g, domain.ErrNotFound
	}
	if err != nil {
		return g, err
	}
	g.ScoreTransferCnt = roundCount
	g.Version = version
	g.MaxParticipants, err = nullStringIntPtr(maxParticipants)
	if err != nil {
		return g, err
	}
	g.PublicShareToken = nullStringPtr(publicShareToken)
	g.LastScoredAt, err = nullTimePtr(lastScoredAt)
	if err != nil {
		return g, err
	}
	g.SettledAt, err = nullTimePtr(settledAt)
	if err != nil {
		return g, err
	}
	g.VoidedAt, err = nullTimePtr(voidedAt)
	if err != nil {
		return g, err
	}
	g.CreatedAt, err = decodeTime(createdAt)
	if err != nil {
		return g, err
	}
	g.UpdatedAt, err = decodeTime(updatedAt)
	return g, err
}

// nullable helpers

func nullStringPtr(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	return &v.String
}

func nullStringIntPtr(v sql.NullString) (*int, error) {
	if !v.Valid {
		return nil, nil
	}
	n := 0
	_, err := fmt.Sscanf(v.String, "%d", &n)
	if err != nil {
		return nil, fmt.Errorf("invalid integer %q: %w", v.String, err)
	}
	return &n, nil
}

func nullTimePtr(v sql.NullString) (*time.Time, error) {
	if !v.Valid {
		return nil, nil
	}
	t, err := decodeTime(v.String)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func nullIntToSQL(v *int) sql.NullInt64 {
	if v == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(*v), Valid: true}
}

func nullStringToSQL(v *string) sql.NullString {
	if v == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *v, Valid: true}
}

func nullTimeToSQL(v *time.Time) sql.NullString {
	if v == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: encodeTime(*v), Valid: true}
}
