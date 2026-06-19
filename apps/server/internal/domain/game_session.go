package domain

import "time"

type GameSessionStatus string

const (
	GameSessionStatusActive   GameSessionStatus = "active"
	GameSessionStatusFinished GameSessionStatus = "finished"
)

type GameSession struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	InviteCode      string            `json:"inviteCode"`
	OwnerUserID     string            `json:"ownerUserId"`
	Status          GameSessionStatus `json:"status"`
	ZeroSumRequired bool              `json:"zeroSumRequired"`
	RoundCount      int               `json:"roundCount"`
	Version         int64             `json:"version"`
	CreatedAt       time.Time         `json:"createdAt"`
	UpdatedAt       time.Time         `json:"updatedAt"`
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
