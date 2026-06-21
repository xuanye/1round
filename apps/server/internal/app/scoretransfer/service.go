package scoretransfer

import (
	"context"
	"time"

	"github.com/google/uuid"
	gamesvc "github.com/xuanye/one-round/apps/server/internal/app/game"
	"github.com/xuanye/one-round/apps/server/internal/app/roundcycle"
	"github.com/xuanye/one-round/apps/server/internal/domain"
	"github.com/xuanye/one-round/apps/server/internal/infra/sqlite"
	"github.com/xuanye/one-round/apps/server/internal/realtime"
)

type Service struct {
	store      *sqlite.Store
	q          *sqlite.Queries
	game       *gamesvc.Service
	roundCycle *roundcycle.Service
	hub        realtime.Hub
	now        func() time.Time
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

func NewService(store *sqlite.Store, q *sqlite.Queries, game *gamesvc.Service, roundCycle *roundcycle.Service, hub realtime.Hub, now func() time.Time) *Service {
	return &Service{store: store, q: q, game: game, roundCycle: roundCycle, hub: hub, now: now}
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

		txRoundCycle := s.roundCycle.WithTx(q)

		// Ensure active round cycle
		round, err := txRoundCycle.EnsureActiveRound(ctx, gameSessionID)
		if err != nil {
			return err
		}

		transfer := domain.ScoreTransfer{
			ID:                   transferID,
			GameSessionID:        gameSessionID,
			RoundCycleID:         round.ID,
			SequenceNo:           sequenceNo,
			FromPlayerID:         sender.ID,
			CreatedByUserID:      userID,
			IdempotencyKey:       input.IdempotencyKey,
			Amount:               input.Amount,
			Kind:                 domain.ScoreTransferKindNormal,
			CreatedAt:            now,
			Receivers:            make([]domain.ScoreTransferReceiver, 0, receiverCount),
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

		// Mark sender and active receivers satisfied for the round
		if err := txRoundCycle.MarkParticipantsSatisfied(ctx, round.ID, transfer.ID, append([]string{sender.ID}, input.ReceiverPlayerIDs...)); err != nil {
			return err
		}
		allSatisfied, err := txRoundCycle.RoundComplete(ctx, round.ID)
		if err != nil {
			return err
		}
		if allSatisfied {
			if err := txRoundCycle.Complete(ctx, round.ID); err != nil {
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

type ReverseInput struct {
	IdempotencyKey string
	Reason         string
}

type ReverseResult struct {
	ID         string `json:"id"`
	SequenceNo int    `json:"sequenceNo"`
	Version    int64  `json:"version"`
}

func (s *Service) Reverse(ctx context.Context, userID, gameSessionID, transferID string, input ReverseInput) (ReverseResult, error) {
	if input.IdempotencyKey == "" {
		return ReverseResult{}, domain.ErrIdempotencyKeyRequired
	}

	// Load active game; reject finished/voided
	session, err := s.q.GetGameSession(ctx, gameSessionID)
	if err != nil {
		return ReverseResult{}, err
	}
	if session.Status != domain.GameSessionStatusActive {
		return ReverseResult{}, domain.ErrGameSessionFinished
	}

	// Load active player for caller; caller must be a current participant
	callerPlayer, err := s.q.GetActivePlayerByUser(ctx, gameSessionID, userID)
	if err != nil {
		if err == domain.ErrNotFound {
			return ReverseResult{}, domain.ErrParticipantRequired
		}
		return ReverseResult{}, err
	}

	now := s.now()
	reversalID := uuid.NewString()
	var sequenceNo int
	var newVersion int64

	// Idempotency check: check if the reversal was already submitted
	existing, err := s.q.GetScoreTransferByIdempotencyKey(ctx, gameSessionID, userID, input.IdempotencyKey)
	if err != nil && err != domain.ErrNotFound {
		return ReverseResult{}, err
	}
	if err == nil {
		// Verify payload
		if existing.Kind != domain.ScoreTransferKindReversal || *existing.ReversalOfTransferID != transferID {
			return ReverseResult{}, domain.ErrIdempotencyConflict
		}
		return ReverseResult{ID: existing.ID, SequenceNo: existing.SequenceNo, Version: session.Version}, nil
	}

	err = s.store.InTx(ctx, func(q *sqlite.Queries) error {
		// Re-validate game status inside transaction
		currentSession, err := q.GetGameSession(ctx, gameSessionID)
		if err != nil {
			return err
		}
		if currentSession.Status != domain.GameSessionStatusActive {
			return domain.ErrGameSessionFinished
		}

		// Re-validate caller
		currentCaller, err := q.GetActivePlayerByUser(ctx, gameSessionID, userID)
		if err != nil {
			if err == domain.ErrNotFound {
				return domain.ErrParticipantRequired
			}
			return err
		}
		if currentCaller.ID != callerPlayer.ID {
			return domain.ErrParticipantRequired
		}

		// Load original transfer with lock
		original, err := q.GetScoreTransferForUpdate(ctx, gameSessionID, transferID)
		if err != nil {
			return err
		}
		if original.ReversedAt != nil {
			return domain.ErrConflict // already reversed
		}
		if original.Kind == domain.ScoreTransferKindReversal {
			return domain.ErrConflict // cannot reverse a reversal
		}

		allowed := false
		for _, r := range original.Receivers {
			if r.PlayerID == callerPlayer.ID {
				allowed = true
				break
			}
		}
		if !allowed {
			return domain.ErrForbidden
		}

		// Allocate sequence number inside transaction
		sequenceNo, err = q.NextScoreTransferSequence(ctx, gameSessionID)
		if err != nil {
			return err
		}

		reversal := domain.ScoreTransfer{
			ID:                   reversalID,
			GameSessionID:        gameSessionID,
			RoundCycleID:         original.RoundCycleID,
			SequenceNo:           sequenceNo,
			FromPlayerID:         original.FromPlayerID,
			CreatedByUserID:      userID,
			IdempotencyKey:       input.IdempotencyKey,
			Amount:               original.Amount,
			Kind:                 domain.ScoreTransferKindReversal,
			ReversalOfTransferID: &original.ID,
			CreatedAt:            now,
			Receivers:            make([]domain.ScoreTransferReceiver, 0, len(original.Receivers)),
		}
		for i, r := range original.Receivers {
			reversal.Receivers = append(reversal.Receivers, domain.ScoreTransferReceiver{
				ID:              uuid.NewString(),
				ScoreTransferID: reversalID,
				PlayerID:        r.PlayerID,
				ReceiverOrder:   i + 1,
			})
		}

		// Insert reversal transfer record
		if err := q.InsertScoreTransferRaw(ctx, reversal); err != nil {
			return err
		}

		// Mark original transfer as reversed
		if err := q.MarkScoreTransferReversed(ctx, original.ID, now); err != nil {
			return err
		}

		// Credit original sender (FromPlayerID)
		totalCredit := original.Amount * len(original.Receivers)
		if err := q.CreditPlayerScore(ctx, gameSessionID, original.FromPlayerID, totalCredit, now); err != nil {
			return err
		}

		// Debit each original receiver
		for _, r := range original.Receivers {
			if err := q.DebitPlayerScore(ctx, gameSessionID, r.PlayerID, original.Amount, now); err != nil {
				return err
			}
		}

		// Reopen the round cycle
		txRoundCycle := s.roundCycle.WithTx(q)
		if err := txRoundCycle.ReopenFromReversal(ctx, original.RoundCycleID, original.ID); err != nil {
			return err
		}

		// Increment round_count, version, updated_at
		newVersion, err = q.IncrementGameSessionForTransfer(ctx, gameSessionID, now)
		return err
	})
	if err != nil {
		return ReverseResult{}, err
	}

	// Broadcast event
	if s.hub != nil {
		s.hub.BroadcastToGame(ctx, gameSessionID, realtime.Event{
			Type:          realtime.EventScoreTransferSubmitted, // We reuse this event so the client reloads everything
			GameSessionID: gameSessionID,
			Version:       newVersion,
			Payload:       map[string]any{"transferId": reversalID, "sequenceNo": sequenceNo},
			SentAt:        now,
		})
	}

	return ReverseResult{ID: reversalID, SequenceNo: sequenceNo, Version: newVersion}, nil
}
