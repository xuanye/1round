package dto

import roundsvc "github.com/xuanye/one-round/apps/server/internal/app/round"

type SubmitRoundRequest struct {
	Scores []roundsvc.ScoreInput `json:"scores"`
	Note   *string               `json:"note"`
}
