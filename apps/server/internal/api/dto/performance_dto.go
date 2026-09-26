package dto

import "time"

type Performance struct {
	TotalScore  int                 `json:"totalScore"`
	TotalGames  int                 `json:"totalGames"`
	Wins        int                 `json:"wins"`
	MaxScore    int                 `json:"maxScore"`
	Trend       []PerformancePoint  `json:"trend"`
	Players     []PerformancePlayer `json:"players"`
	RecentGames []PerformanceGame   `json:"recentGames"`
}
type PerformancePoint struct {
	SettledAt time.Time `json:"settledAt"`
	Score     int       `json:"score"`
}
type PerformancePlayer struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	AvatarURL   string `json:"avatarUrl"`
	TotalScore  int    `json:"totalScore"`
	IsMe        bool   `json:"isMe"`
}
type PerformanceGame struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	SettledAt        time.Time `json:"settledAt"`
	ParticipantCount int       `json:"participantCount"`
	MyFinalScore     int       `json:"myFinalScore"`
}
