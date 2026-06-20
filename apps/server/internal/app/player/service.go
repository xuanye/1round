package player

import (
	"context"
	"strings"
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

func NewService(store *sqlite.Store, q *sqlite.Queries, game *gamesvc.Service, hub realtime.Hub, now func() time.Time) *Service {
	return &Service{store: store, q: q, game: game, hub: hub, now: now}
}

func (s *Service) Add(ctx context.Context, userID, gameSessionID, displayName string) (domain.Player, error) {
	if strings.TrimSpace(displayName) == "" {
		return domain.Player{}, domain.ErrInvalidArgument
	}
	if err := s.game.RequireMember(ctx, userID, gameSessionID); err != nil {
		return domain.Player{}, err
	}
	session, err := s.q.GetGameSession(ctx, gameSessionID)
	if err != nil {
		return domain.Player{}, err
	}
	if session.Status == domain.GameSessionStatusFinished {
		return domain.Player{}, domain.ErrGameSessionFinished
	}
	now := s.now()
	joinedOrder, err := s.q.NextJoinedOrder(ctx, gameSessionID)
	if err != nil {
		return domain.Player{}, err
	}
	p := domain.Player{ID: uuid.NewString(), GameSessionID: gameSessionID, DisplayName: strings.TrimSpace(displayName), Active: true, JoinedOrder: joinedOrder, CreatedAt: now, UpdatedAt: now}
	err = s.q.CreatePlayer(ctx, p)
	if err == nil && s.hub != nil {
		s.hub.BroadcastToGame(ctx, gameSessionID, realtime.Event{Type: realtime.EventPlayerAdded, GameSessionID: gameSessionID, Version: session.Version, SentAt: s.now()})
	}
	return p, err
}

func (s *Service) List(ctx context.Context, userID, gameSessionID string) ([]domain.Player, error) {
	if err := s.game.RequireMember(ctx, userID, gameSessionID); err != nil {
		return nil, err
	}
	return s.q.ListPlayers(ctx, gameSessionID)
}

func (s *Service) Update(ctx context.Context, userID, gameSessionID, playerID, displayName string) (domain.Player, error) {
	if err := s.game.RequireMember(ctx, userID, gameSessionID); err != nil {
		return domain.Player{}, err
	}
	return s.q.UpdatePlayer(ctx, gameSessionID, playerID, strings.TrimSpace(displayName))
}

func (s *Service) Delete(ctx context.Context, userID, gameSessionID, playerID string) error {
	if err := s.game.RequireMember(ctx, userID, gameSessionID); err != nil {
		return err
	}
	return s.q.DeletePlayer(ctx, gameSessionID, playerID)
}

func (s *Service) UpdateMyProfile(ctx context.Context, userID, gameSessionID, displayName string) (domain.Player, error) {
	if strings.TrimSpace(displayName) == "" {
		return domain.Player{}, domain.ErrInvalidArgument
	}
	session, err := s.q.GetGameSession(ctx, gameSessionID)
	if err != nil {
		return domain.Player{}, err
	}
	if session.Status == domain.GameSessionStatusFinished || session.Status == domain.GameSessionStatusVoided {
		return domain.Player{}, domain.ErrGameSessionFinished
	}

	// User must be a historical participant
	player, err := s.q.GetHistoricalPlayerByUser(ctx, gameSessionID, userID)
	if err != nil {
		return domain.Player{}, err
	}

	displayName = strings.TrimSpace(displayName)

	// Check display name uniqueness excluding self
	historicalPlayers, err := s.q.ListHistoricalPlayers(ctx, gameSessionID)
	if err != nil {
		return domain.Player{}, err
	}
	for _, p := range historicalPlayers {
		if p.ID == player.ID {
			continue
		}
		if p.DisplayName == displayName {
			return domain.Player{}, domain.ErrDuplicateDisplayName
		}
	}

	now := s.now()
	var updated domain.Player
	err = s.store.InTx(ctx, func(q *sqlite.Queries) error {
		var txErr error
		updated, txErr = q.UpdatePlayer(ctx, gameSessionID, player.ID, displayName)
		if txErr != nil {
			return txErr
		}
		_, txErr = q.UpdateUserDisplayName(ctx, userID, &displayName, now)
		return txErr
	})
	if err != nil {
		return domain.Player{}, err
	}
	if s.hub != nil {
		s.hub.BroadcastToGame(ctx, gameSessionID, realtime.Event{Type: realtime.EventParticipantUpdated, GameSessionID: gameSessionID, Version: session.Version, SentAt: now})
	}
	return updated, nil
}

func (s *Service) Leave(ctx context.Context, userID, gameSessionID string) error {
	// Load active session
	session, err := s.q.GetGameSession(ctx, gameSessionID)
	if err != nil {
		return err
	}
	if session.Status != domain.GameSessionStatusActive {
		return domain.ErrGameSessionFinished
	}

	// Verify user is a game member before proceeding
	if err := s.game.RequireMember(ctx, userID, gameSessionID); err != nil {
		return err
	}

	// Load active player for user
	player, err := s.q.GetActivePlayerByUser(ctx, gameSessionID, userID)
	if err != nil {
		return err
	}

	// If TotalScore != 0, reject
	if player.TotalScore != 0 {
		return domain.ErrCannotLeaveWithNonZeroScore
	}

	now := s.now()
	isOwner := session.OwnerUserID == userID
	var activeCount int

	err = s.store.InTx(ctx, func(q *sqlite.Queries) error {
		// Mark player inactive and set left_at
		if err := q.DeactivatePlayer(ctx, gameSessionID, userID, now); err != nil {
			return err
		}

		// If leaving owner, transfer ownership to the next eligible player.
		// GetNextOwner filters for active players with a non-nil UserID,
		// but we guard nil UserID here as defense-in-depth.
		if isOwner {
			nextOwner, err := q.GetNextOwner(ctx, gameSessionID)
			if err != nil {
				return err
			}
			if nextOwner != nil {
				uid := nextOwner.UserID
				if uid == nil {
					// No eligible owner found with a linked user; ownership
					// cannot be transferred. The game retains the stale owner
					// until an admin intervenes or the game finishes.
					return domain.ErrOwnerRequired
				}
				if err := q.UpdateGameSessionOwner(ctx, gameSessionID, *uid, now); err != nil {
					return err
				}
				if err := q.TransferGameMemberRole(ctx, gameSessionID, *uid, now); err != nil {
					return err
				}
			}
		}

		// Count remaining active players
		activeCount, err = q.CountActiveParticipants(ctx, gameSessionID)
		if err != nil {
			return err
		}

		if activeCount == 0 {
			// No active players remain
			if session.ScoreTransferCnt == 0 {
				// Void the game
				if _, err := q.VoidGameSession(ctx, gameSessionID, now); err != nil {
					return err
				}
			} else {
				// Finish the game
				shareToken := uuid.NewString()
				if _, err := q.FinishGameSessionWithSettle(ctx, gameSessionID, now); err != nil {
					return err
				}
				if err := q.SetPublicShareToken(ctx, gameSessionID, shareToken); err != nil {
					return err
				}
			}
		} else {
			// Active players remain - just increment version
			if err := q.IncrementGameSessionVersion(ctx, gameSessionID, now); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Broadcast events after successful transaction
	if activeCount == 0 {
		if session.ScoreTransferCnt == 0 {
			if s.hub != nil {
				s.hub.BroadcastToGame(ctx, gameSessionID, realtime.Event{Type: realtime.EventGameVoided, GameSessionID: gameSessionID, Version: session.Version + 1, SentAt: now})
			}
		} else {
			if s.hub != nil {
				s.hub.BroadcastToGame(ctx, gameSessionID, realtime.Event{Type: realtime.EventGameFinished, GameSessionID: gameSessionID, Version: session.Version + 1, SentAt: now})
			}
		}
	} else {
		if s.hub != nil {
			s.hub.BroadcastToGame(ctx, gameSessionID, realtime.Event{Type: realtime.EventParticipantLeft, GameSessionID: gameSessionID, Version: session.Version + 1, SentAt: now})
		}
	}

	return nil
}
