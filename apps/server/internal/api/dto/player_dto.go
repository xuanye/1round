package dto

type PlayerRequest struct {
	DisplayName string `json:"displayName"`
}

type DeletePlayerResponse struct {
	Deleted bool `json:"deleted"`
}
