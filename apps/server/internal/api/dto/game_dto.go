package dto

import "time"

type CreateGameRequest struct {
	Name            string `json:"name"`
	MaxParticipants *int   `json:"maxParticipants"`
	PresetScores    []int  `json:"presetScores"`
}

type JoinGameRequest struct {
	InviteCode  string `json:"inviteCode"`
	DisplayName string `json:"displayName"`
}

type JoinPreviewRequest struct {
	InviteCode string `json:"inviteCode"`
}

type CurrentGameResponse struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	InviteCode       string    `json:"inviteCode"`
	OwnerUserID      string    `json:"ownerUserId"`
	Status           string    `json:"status"`
	MaxParticipants  *int      `json:"maxParticipants"`
	PresetScores     []int     `json:"presetScores"`
	ScoreTransferCnt int       `json:"scoreTransferCount"`
	Version          int64     `json:"version"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}
