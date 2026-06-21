package roundcycle

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/xuanye/one-round/apps/server/internal/domain"
	"github.com/xuanye/one-round/apps/server/internal/infra/sqlite"
)

type Service struct {
	q   *sqlite.Queries
	now func() time.Time
}

func NewService(q *sqlite.Queries, now func() time.Time) *Service {
	return &Service{q: q, now: now}
}

func (s *Service) WithTx(q *sqlite.Queries) *Service {
	return &Service{q: q, now: s.now}
}

func (s *Service) EnsureActiveRound(ctx context.Context, gameSessionID string) (domain.RoundCycle, error) {
	rc, err := s.q.GetActiveRoundCycle(ctx, gameSessionID)
	if err == nil {
		return rc, nil
	}
	if err != domain.ErrNotFound {
		return domain.RoundCycle{}, err
	}

	last, err := s.q.GetLatestRoundCycle(ctx, gameSessionID)
	if err != nil {
		return domain.RoundCycle{}, err
	}

	now := s.now()
	next := domain.RoundCycle{
		ID:            uuid.NewString(),
		GameSessionID: gameSessionID,
		RoundNo:       last.RoundNo + 1,
		Status:        domain.RoundCycleStatusActive,
		CreatedAt:     now,
	}
	if err := s.q.CreateRoundCycle(ctx, next); err != nil {
		return domain.RoundCycle{}, err
	}

	// Add all currently active players of the session as pending to this round
	players, err := s.q.ListActivePlayers(ctx, gameSessionID)
	if err != nil {
		return domain.RoundCycle{}, err
	}
	for _, p := range players {
		err := s.q.UpsertRoundParticipationStatus(ctx, domain.RoundParticipationStatus{
			ID:           uuid.NewString(),
			RoundCycleID: next.ID,
			PlayerID:     p.ID,
			Status:       domain.ParticipationStatusPending,
			UpdatedAt:    now,
		})
		if err != nil {
			return domain.RoundCycle{}, err
		}
	}
	return next, nil
}

func (s *Service) MarkParticipantsSatisfied(ctx context.Context, roundCycleID, transferID string, playerIDs []string) error {
	now := s.now()
	for _, pid := range playerIDs {
		// Update status to satisfied and record the transfer ID
		err := s.q.UpsertRoundParticipationStatus(ctx, domain.RoundParticipationStatus{
			ID:                  uuid.NewString(),
			RoundCycleID:        roundCycleID,
			PlayerID:            pid,
			Status:              domain.ParticipationStatusSatisfied,
			SatisfiedByTransfer: &transferID,
			UpdatedAt:           now,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) RoundComplete(ctx context.Context, roundCycleID string) (bool, error) {
	statuses, err := s.q.ListRoundParticipationStatuses(ctx, roundCycleID)
	if err != nil {
		return false, err
	}
	for _, status := range statuses {
		if status.Status == domain.ParticipationStatusPending {
			return false, nil
		}
	}
	return true, nil
}

func (s *Service) Complete(ctx context.Context, roundCycleID string) error {
	return s.q.CompleteRoundCycle(ctx, roundCycleID, s.now())
}

func (s *Service) ReopenFromReversal(ctx context.Context, roundCycleID, transferID string) error {
	now := s.now()
	if err := s.q.ResetRoundParticipationAfterReversal(ctx, roundCycleID, transferID, now); err != nil {
		return err
	}
	return s.q.ReopenRoundCycle(ctx, roundCycleID)
}
