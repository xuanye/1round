package dto

type SubmitScoreTransferRequest struct {
	ReceiverPlayerIDs []string `json:"receiverPlayerIds"`
	Amount            int      `json:"amount"`
	IdempotencyKey    string   `json:"idempotencyKey"`
}

type ReverseScoreTransferRequest struct {
	IdempotencyKey string `json:"idempotencyKey"`
	Reason         string `json:"reason"`
}
