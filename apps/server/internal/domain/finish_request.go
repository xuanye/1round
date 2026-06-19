package domain

import "time"

type FinishRequestStatus string

const (
	FinishRequestStatusPending  FinishRequestStatus = "pending"
	FinishRequestStatusApproved FinishRequestStatus = "approved"
	FinishRequestStatusRejected FinishRequestStatus = "rejected"
)

type FinishRequest struct {
	ID                  string
	GameSessionID       string
	RequestedByPlayerID string
	Status              FinishRequestStatus
	CreatedAt           time.Time
	DecidedAt           *time.Time
	DecidedByPlayerID   *string
}
