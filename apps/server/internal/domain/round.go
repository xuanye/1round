package domain

import "time"

type Round struct {
	ID              string
	GameSessionID   string
	RoundNo         int
	CreatedByUserID string
	Note            *string
	CreatedAt       time.Time
	Scores          []RoundScore
}

type RoundScore struct {
	ID       string
	RoundID  string
	PlayerID string
	Score    int
}
