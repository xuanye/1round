package domain

import "time"

type User struct {
	ID          string
	OpenID      string
	UnionID     *string
	DisplayName *string
	AvatarURL   *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
