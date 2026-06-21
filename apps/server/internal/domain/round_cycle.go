package domain

import "time"

type RoundCycleStatus string

const (
	RoundCycleStatusActive   RoundCycleStatus = "active"
	RoundCycleStatusComplete RoundCycleStatus = "complete"
)

type ParticipationStatus string

const (
	ParticipationStatusPending   ParticipationStatus = "pending"
	ParticipationStatusSatisfied ParticipationStatus = "satisfied"
	ParticipationStatusExempt    ParticipationStatus = "exempt"
)

type RoundCycle struct {
	ID            string
	GameSessionID string
	RoundNo       int
	Status        RoundCycleStatus
	CreatedAt     time.Time
	CompletedAt   *time.Time
}

type RoundParticipationStatus struct {
	ID                  string
	RoundCycleID        string
	PlayerID            string
	Status              ParticipationStatus
	SatisfiedByTransfer *string
	UpdatedAt           time.Time
}
