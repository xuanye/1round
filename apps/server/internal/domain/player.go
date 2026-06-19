package domain

import "time"

type Player struct {
	ID            string    `json:"id"`
	GameSessionID string    `json:"gameSessionId"`
	UserID        *string   `json:"userId"`
	DisplayName   string    `json:"displayName"`
	TotalScore    int       `json:"totalScore"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}
