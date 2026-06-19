package domain

import "time"

type ScoreTransfer struct {
	ID              string
	GameSessionID   string
	SequenceNo      int
	FromPlayerID    string
	CreatedByUserID string
	IdempotencyKey  string
	Amount          int
	CreatedAt       time.Time
	Receivers       []ScoreTransferReceiver
}

type ScoreTransferReceiver struct {
	ID              string
	ScoreTransferID string
	PlayerID        string
	ReceiverOrder   int
}

func ValidateScoreTransferInput(amount int, receiverPlayerIDs []string) error {
	if amount <= 0 {
		return ErrInvalidScoreTransferAmount
	}
	if len(receiverPlayerIDs) == 0 {
		return ErrScoreTransferReceiverRequired
	}
	seen := map[string]struct{}{}
	for _, id := range receiverPlayerIDs {
		if id == "" {
			return ErrInvalidPlayer
		}
		if _, ok := seen[id]; ok {
			return ErrInvalidPlayer
		}
		seen[id] = struct{}{}
	}
	return nil
}
