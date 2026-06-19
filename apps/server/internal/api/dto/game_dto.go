package dto

type CreateGameRequest struct {
	Name            string `json:"name"`
	MaxParticipants *int   `json:"maxParticipants"`
}

type JoinGameRequest struct {
	InviteCode string `json:"inviteCode"`
}
