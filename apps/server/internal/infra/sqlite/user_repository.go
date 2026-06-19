package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/xuanye/one-round/apps/server/internal/domain"
)

func (q *Queries) UpsertUserByOpenID(ctx context.Context, openID string, now time.Time) (domain.User, error) {
	var u domain.User
	var createdAt, updatedAt string
	err := q.db.QueryRowContext(ctx, `SELECT id, open_id, union_id, display_name, avatar_url, created_at, updated_at FROM users WHERE open_id = ?`, openID).
		Scan(&u.ID, &u.OpenID, &u.UnionID, &u.DisplayName, &u.AvatarURL, &createdAt, &updatedAt)
	if err == nil {
		u.CreatedAt, _ = decodeTime(createdAt)
		u.UpdatedAt, _ = decodeTime(updatedAt)
		return u, nil
	}
	if err != sql.ErrNoRows {
		return u, err
	}
	u = domain.User{ID: uuid.NewString(), OpenID: openID, CreatedAt: now, UpdatedAt: now}
	_, err = q.db.ExecContext(ctx, `INSERT INTO users (id, open_id, created_at, updated_at) VALUES (?, ?, ?, ?)`, u.ID, u.OpenID, encodeTime(u.CreatedAt), encodeTime(u.UpdatedAt))
	return u, err
}
