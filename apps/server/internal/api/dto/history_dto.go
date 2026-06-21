package dto

import "time"

type HistoryPage struct {
	Items      []HistoryItem `json:"items"`
	NextCursor *string       `json:"nextCursor,omitempty"`
}

type HistoryItem struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	SettledAt          time.Time `json:"settledAt"`
	ScoreTransferCount int       `json:"scoreTransferCount"`
	MyFinalScore       int       `json:"myFinalScore"`
	ParticipantCount   int       `json:"participantCount"`
	WinnerName         string    `json:"winnerName"`
	WinnerScore        int       `json:"winnerScore"`
	CreatedAt          time.Time `json:"createdAt"`
}

type SettlementDetail struct {
	ID               string                  `json:"id"`
	Name             string                  `json:"name"`
	SettledAt        time.Time               `json:"settledAt"`
	Participants     []SettlementParticipant `json:"participants"`
	ScoreTransfers   []ScoreTransferSummary  `json:"scoreTransfers"`
	NextCursor       *int                    `json:"nextCursor,omitempty"`
	PublicShareToken *string                 `json:"publicShareToken,omitempty"`
}

type SettlementParticipant struct {
	ID          string  `json:"id"`
	DisplayName string  `json:"displayName"`
	AvatarURL   *string `json:"avatarUrl,omitempty"`
	FinalScore  int     `json:"finalScore"`
}

type ScoreTransferSummary struct {
	ID                   string     `json:"id"`
	SequenceNo           int        `json:"sequenceNo"`
	Amount               int        `json:"amount"`
	CreatedAt            string     `json:"createdAt"`
	Text                 string     `json:"text"`
	TransferKind         string     `json:"transferKind,omitempty"`
	ReversalOfTransferID *string    `json:"reversalOfTransferId,omitempty"`
	ReversedAt           *time.Time `json:"reversedAt,omitempty"`
}

type PublicSettlement struct {
	GameSessionID  string                  `json:"gameSessionId"`
	Name           string                  `json:"name"`
	SettledAt      time.Time               `json:"settledAt"`
	Participants   []SettlementParticipant `json:"participants"`
	ScoreTransfers []ScoreTransferSummary  `json:"scoreTransfers"`
}

type UserStatsResponse struct {
	TotalGames int `json:"totalGames"`
	MaxScore   int `json:"maxScore"`
}

