package domain

import "time"

type GameSessionStatus string

const (
	GameSessionStatusActive   GameSessionStatus = "active"
	GameSessionStatusFinished GameSessionStatus = "finished"
	GameSessionStatusVoided   GameSessionStatus = "voided"
)

type GameSession struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	InviteCode       string            `json:"inviteCode"`
	OwnerUserID      string            `json:"ownerUserId"`
	Status           GameSessionStatus `json:"status"`
	MaxParticipants  *int              `json:"maxParticipants"`
	ScoreTransferCnt int               `json:"scoreTransferCount"`
	Version          int64             `json:"version"`
	PublicShareToken *string           `json:"publicShareToken,omitempty"`
	LastScoredAt     *time.Time        `json:"lastScoredAt,omitempty"`
	SettledAt        *time.Time        `json:"settledAt,omitempty"`
	VoidedAt         *time.Time        `json:"voidedAt,omitempty"`
	CreatedAt        time.Time         `json:"createdAt"`
	UpdatedAt        time.Time         `json:"updatedAt"`
}

type GameMemberRole string

const (
	GameMemberRoleOwner  GameMemberRole = "owner"
	GameMemberRoleMember GameMemberRole = "member"
)

type GameMember struct {
	ID            string
	GameSessionID string
	UserID        string
	Role          GameMemberRole
	JoinedAt      time.Time
}
