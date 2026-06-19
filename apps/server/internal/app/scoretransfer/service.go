package scoretransfer

import (
	"context"
	"time"

	"github.com/google/uuid"
	gamesvc "github.com/xuanye/one-round/apps/server/internal/app/game"
	"github.com/xuanye/one-round/apps/server/internal/domain"
	"github.com/xuanye/one-round/apps/server/internal/infra/sqlite"
	"github.com/xuanye/one-round/apps/server/internal/realtime"
)

type Service struct {
	store *sqlite.Store
	q     *sqlite.Queries
	game  *gamesvc.Service
	hub   realtime.Hub
	now   func() time.Time
}

type SubmitInput struct {
	ReceiverPlayerIDs []string
	Amount            int
	IdempotencyKey    string
}

type SubmitResult struct {
	ID         string `json:"id"`
	SequenceNo int    `json:"sequenceNo"`
	Version    int64  `json:"version"`
}

func NewService(store *sqlite.Store, q *sqlite.Queries, game *gamesvc.Service, hub realtime.Hub, now func() time.Time) *Service {
	return &Service{store: store, q: q, game: game, hub: hub, now: now}
}

func (s *Service) Submit(ctx context.Context, userID, gameSessionID string, input SubmitInput) (SubmitResult, error) {
	if input.IdempotencyKey == "" {
		return SubmitResult{}, domain.ErrIdempotencyKeyRequired
	}
	if err := domain.ValidateScoreTransferInput(input.Amount, input.ReceiverPlayerIDs); err != nil {
		return SubmitResult{}, err
	}

	// Load active game; reject finished/voided
	session, err := s.q.GetGameSession(ctx, gameSessionID)
	if err != nil {
		return SubmitResult{}, err
	}
	if session.Status != domain.GameSessionStatusActive {
		return SubmitResult{}, domain.ErrGameSessionFinished
	}

	// Load active sender by userID; sender must be a current participant
	sender, err := s.q.GetActivePlayerByUser(ctx, gameSessionID, userID)
	if err != nil {
		if err == domain.ErrNotFound {
			return SubmitResult{}, domain.ErrParticipantRequired
		}
		return SubmitResult{}, err
	}

	// Reject any receiver equal to sender
	for _, rid := range input.ReceiverPlayerIDs {
		if rid == sender.ID {
			return SubmitResult{}, domain.ErrInvalidPlayer
		}
	}

	// Load all receiver players as active in the same game
	for _, rid := range input.ReceiverPlayerIDs {
		p, err := s.q.GetPlayer(ctx, gameSessionID, rid)
		if err != nil {
			if err == domain.ErrNotFound {
				return SubmitResult{}, domain.ErrInvalidPlayer
			}
			return SubmitResult{}, err
		}
		if !p.Active {
			return SubmitResult{}, domain.ErrParticipantInactive
		}
	}

	// Idempotency check: look for existing transfer by (gameSessionID, userID, idempotencyKey)
	existing, err := s.q.GetScoreTransferByIdempotencyKey(ctx, gameSessionID, userID, input.IdempotencyKey)
	if err != nil && err != domain.ErrNotFound {
		return SubmitResult{}, err
	}
	if err == nil {
		// Existing transfer found -- verify same payload
		if existing.Amount != input.Amount || len(existing.Receivers) != len(input.ReceiverPlayerIDs) {
			return SubmitResult{}, domain.ErrIdempotencyConflict
		}
		existingReceiverSet := map[string]struct{}{}
		for _, r := range existing.Receivers {
			existingReceiverSet[r.PlayerID] = struct{}{}
		}
		for _, rid := range input.ReceiverPlayerIDs {
			if _, ok := existingReceiverSet[rid]; !ok {
				return SubmitResult{}, domain.ErrIdempotencyConflict
			}
		}
		return SubmitResult{ID: existing.ID, SequenceNo: existing.SequenceNo, Version: session.Version}, nil
	}

	// New transfer: persist in one transaction
	now := s.now()
	transferID := uuid.NewString()
	receiverCount := len(input.ReceiverPlayerIDs)

	var sequenceNo int
	var newVersion int64
	err = s.store.InTx(ctx, func(q *sqlite.Queries) error {
		// Re-validate game status inside transaction to prevent writes after finish
		currentSession, err := q.GetGameSession(ctx, gameSessionID)
		if err != nil {
			return err
		}
		if currentSession.Status != domain.GameSessionStatusActive {
			return domain.ErrGameSessionFinished
		}

		// Re-validate sender is still active
		currentSender, err := q.GetActivePlayerByUser(ctx, gameSessionID, userID)
		if err != nil {
			if err == domain.ErrNotFound {
				return domain.ErrParticipantRequired
			}
			return err
		}
		if currentSender.ID != sender.ID {
			return domain.ErrParticipantRequired
		}

		// Re-validate all receivers are still active
		for _, rid := range input.ReceiverPlayerIDs {
			p, err := q.GetPlayer(ctx, gameSessionID, rid)
			if err != nil {
				if err == domain.ErrNotFound {
					return domain.ErrInvalidPlayer
				}
				return err
			}
			if !p.Active {
				return domain.ErrParticipantInactive
			}
		}

		// Allocate sequence number inside transaction to prevent concurrent conflicts
		sequenceNo, err = q.NextScoreTransferSequence(ctx, gameSessionID)
		if err != nil {
			return err
		}

		transfer := domain.ScoreTransfer{
			ID:              transferID,
			GameSessionID:   gameSessionID,
			SequenceNo:      sequenceNo,
			FromPlayerID:    sender.ID,
			CreatedByUserID: userID,
			IdempotencyKey:  input.IdempotencyKey,
			Amount:          input.Amount,
			CreatedAt:       now,
			Receivers:       make([]domain.ScoreTransferReceiver, 0, receiverCount),
		}
		for i, rid := range input.ReceiverPlayerIDs {
			transfer.Receivers = append(transfer.Receivers, domain.ScoreTransferReceiver{
				ID:              uuid.NewString(),
				ScoreTransferID: transferID,
				PlayerID:        rid,
				ReceiverOrder:   i + 1,
			})
		}

		// Insert the score transfer record
		if err := q.InsertScoreTransferRaw(ctx, transfer); err != nil {
			return err
		}
		// Debit sender
		totalDebit := input.Amount * receiverCount
		if err := q.DebitPlayerScore(ctx, gameSessionID, sender.ID, totalDebit, now); err != nil {
			return err
		}
		// Credit each receiver
		for _, r := range transfer.Receivers {
			if err := q.CreditPlayerScore(ctx, gameSessionID, r.PlayerID, input.Amount, now); err != nil {
				return err
			}
		}
		// Increment round_count, version, updated_at, and last_scored_at using atomic version increment
		newVersion, err = q.IncrementGameSessionForTransfer(ctx, gameSessionID, now)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return SubmitResult{}, err
	}

	// Broadcast event
	if s.hub != nil {
		s.hub.BroadcastToGame(ctx, gameSessionID, realtime.Event{
			Type:          realtime.EventScoreTransferSubmitted,
			GameSessionID: gameSessionID,
			Version:       newVersion,
			Payload:       map[string]any{"transferId": transferID, "sequenceNo": sequenceNo},
			SentAt:        now,
		})
	}

	return SubmitResult{ID: transferID, SequenceNo: sequenceNo, Version: newVersion}, nil
}
