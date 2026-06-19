package dto

type CreateGameRequest struct {
	Name            string `json:"name"`
	ZeroSumRequired bool   `json:"zeroSumRequired"`
}

type JoinGameRequest struct {
	InviteCode string `json:"inviteCode"`
}
