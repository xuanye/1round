package realtime

import "time"

const (
	EventGameUpdated    = "game.updated"
	EventPlayerAdded    = "player.added"
	EventPlayerUpdated  = "player.updated"
	EventPlayerRemoved  = "player.removed"
	EventRoundSubmitted = "round.submitted"
	EventGameFinished          = "game.finished"
	EventGameVoided            = "game.voided"
	EventParticipantJoined     = "participant.joined"
	EventParticipantUpdated    = "participant.updated"
	EventParticipantLeft            = "participant.left"
	EventScoreTransferSubmitted     = "score_transfer.submitted"
)

type Event struct {
	Type          string    `json:"type"`
	GameSessionID string    `json:"gameSessionId"`
	Version       int64     `json:"version"`
	Payload       any       `json:"payload,omitempty"`
	SentAt        time.Time `json:"sentAt"`
}
