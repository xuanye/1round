package settlement

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"log/slog"
	"time"

	"github.com/google/uuid"
	gamesvc "github.com/xuanye/one-round/apps/server/internal/app/game"
	"github.com/xuanye/one-round/apps/server/internal/domain"
	"github.com/xuanye/one-round/apps/server/internal/infra/sqlite"
	"github.com/xuanye/one-round/apps/server/internal/realtime"
)

type Service struct {
	store  *sqlite.Store
	q      *sqlite.Queries
	game   *gamesvc.Service
	hub    realtime.Hub
	now    func() time.Time
	logger *slog.Logger
}

func NewService(store *sqlite.Store, q *sqlite.Queries, game *gamesvc.Service, hub realtime.Hub, now func() time.Time) *Service {
	return &Service{store: store, q: q, game: game, hub: hub, now: now, logger: slog.Default()}
}

func GenerateShareToken() (string, error) {
	buf := make([]byte, 18)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// FinishDirect allows the game owner to finish an active game directly.
// The game is marked finished, settled_at is set, and a public share token is generated.
func (s *Service) FinishDirect(ctx context.Context, userID, gameSessionID string) (domain.GameSession, error) {
	if err := s.game.RequireMember(ctx, userID, gameSessionID); err != nil {
		return domain.GameSession{}, err
	}
	session, err := s.q.GetGameSession(ctx, gameSessionID)
	if err != nil {
		return domain.GameSession{}, err
	}
	if session.Status != domain.GameSessionStatusActive {
		return domain.GameSession{}, domain.ErrGameSessionFinished
	}
	if session.OwnerUserID != userID {
		return domain.GameSession{}, domain.ErrNotOwner
	}

	now := s.now()
	shareToken, err := GenerateShareToken()
	if err != nil {
		return domain.GameSession{}, err
	}
	session, err = s.q.FinishGameSessionWithSettleAndToken(ctx, gameSessionID, shareToken, now)
	if err != nil {
		return domain.GameSession{}, err
	}
	if s.hub != nil {
		s.hub.BroadcastToGame(ctx, gameSessionID, realtime.Event{Type: realtime.EventGameFinished, GameSessionID: gameSessionID, Version: session.Version, SentAt: now})
	}
	return session, nil
}

// InactiveSettlementResult reports the outcome of an auto-settlement scan.
type InactiveSettlementResult struct {
	Finished int `json:"finished"`
	Voided   int `json:"voided"`
}

// SettleInactiveGames scans for active game sessions that have been inactive beyond the
// given threshold and either voids them (if no score transfers) or finishes them (if scored).
// It returns the number of games finished and voided.
func (s *Service) SettleInactiveGames(ctx context.Context, threshold time.Duration) (InactiveSettlementResult, error) {
	now := s.now()
	cutoff := now.Add(-threshold)

	candidates, err := s.q.ListInactiveActiveSessions(ctx, cutoff, 100)
	if err != nil {
		return InactiveSettlementResult{}, err
	}

	var result InactiveSettlementResult
	for _, c := range candidates {
		action, err := s.settleOneInactive(ctx, c.ID, now)
		if err != nil {
			s.logger.Error("auto settlement failed for game", "game_id", c.ID, "error", err)
			continue
		}
		switch action {
		case "finished":
			result.Finished++
		case "voided":
			result.Voided++
		}
	}
	return result, nil
}

// settleOneInactive performs an idempotent settlement for a single game session.
// It re-reads the game within a transaction to verify it is still active before acting.
func (s *Service) settleOneInactive(ctx context.Context, gameSessionID string, now time.Time) (string, error) {
	var action string
	err := s.store.InTx(ctx, func(q *sqlite.Queries) error {
		session, err := q.GetGameSession(ctx, gameSessionID)
		if err != nil {
			return err
		}
		if session.Status != domain.GameSessionStatusActive {
			return nil // already settled/voided by someone else
		}

		if session.ScoreTransferCnt == 0 {
			// Void the game
			_, err = q.VoidGameSession(ctx, gameSessionID, now)
			if err != nil {
				return err
			}
			action = "voided"
			return nil
		}

		// Finish the game with a share token
		shareToken, err := GenerateShareToken()
		if err != nil {
			return err
		}
		_, err = q.FinishGameSessionWithSettleAndToken(ctx, gameSessionID, shareToken, now)
		if err != nil {
			return err
		}
		action = "finished"
		return nil
	})
	if err != nil {
		return "", err
	}

	// Broadcast events outside the transaction
	if s.hub != nil && action != "" {
		session, err := s.q.GetGameSession(ctx, gameSessionID)
		if err == nil {
			switch action {
			case "finished":
				s.hub.BroadcastToGame(ctx, gameSessionID, realtime.Event{Type: realtime.EventGameFinished, GameSessionID: gameSessionID, Version: session.Version, SentAt: now})
			case "voided":
				s.hub.BroadcastToGame(ctx, gameSessionID, realtime.Event{Type: realtime.EventGameVoided, GameSessionID: gameSessionID, Version: session.Version, SentAt: now})
			}
		}
	}

	return action, nil
}

// RequestFinish creates a pending finish request from a non-owner participant.
// Only one pending request may exist at a time.
func (s *Service) RequestFinish(ctx context.Context, userID, gameSessionID string) (domain.FinishRequest, error) {
	if err := s.game.RequireMember(ctx, userID, gameSessionID); err != nil {
		return domain.FinishRequest{}, err
	}
	session, err := s.q.GetGameSession(ctx, gameSessionID)
	if err != nil {
		return domain.FinishRequest{}, err
	}
	if session.Status != domain.GameSessionStatusActive {
		return domain.FinishRequest{}, domain.ErrGameSessionFinished
	}
	if session.OwnerUserID == userID {
		return domain.FinishRequest{}, domain.ErrOwnerRequired
	}

	// Check no pending request already exists
	pending, err := s.q.GetPendingFinishRequest(ctx, gameSessionID)
	if err != nil {
		return domain.FinishRequest{}, err
	}
	if pending != nil {
		return domain.FinishRequest{}, domain.ErrFinishRequestPending
	}

	// Get the user's player record
	player, err := s.q.GetActivePlayerByUser(ctx, gameSessionID, userID)
	if err != nil {
		return domain.FinishRequest{}, err
	}

	now := s.now()
	req := domain.FinishRequest{
		ID:                  uuid.NewString(),
		GameSessionID:       gameSessionID,
		RequestedByPlayerID: player.ID,
		Status:              domain.FinishRequestStatusPending,
		CreatedAt:           now,
	}
	err = s.q.CreateFinishRequest(ctx, req)
	if err != nil {
		return domain.FinishRequest{}, err
	}
	if s.hub != nil {
		s.hub.BroadcastToGame(ctx, gameSessionID, realtime.Event{Type: realtime.EventFinishRequested, GameSessionID: gameSessionID, Version: session.Version, SentAt: now})
	}
	return req, nil
}

// ApproveFinishRequest allows the game owner to approve a pending finish request.
// This finishes the game and generates a share token.
func (s *Service) ApproveFinishRequest(ctx context.Context, userID, gameSessionID, requestID string) (domain.GameSession, error) {
	if err := s.game.RequireMember(ctx, userID, gameSessionID); err != nil {
		return domain.GameSession{}, err
	}
	session, err := s.q.GetGameSession(ctx, gameSessionID)
	if err != nil {
		return domain.GameSession{}, err
	}
	if session.Status != domain.GameSessionStatusActive {
		return domain.GameSession{}, domain.ErrGameSessionFinished
	}
	if session.OwnerUserID != userID {
		return domain.GameSession{}, domain.ErrNotOwner
	}

	req, err := s.q.GetFinishRequest(ctx, requestID)
	if err != nil {
		return domain.GameSession{}, err
	}
	if req.GameSessionID != gameSessionID {
		return domain.GameSession{}, domain.ErrNotFound
	}
	if req.Status != domain.FinishRequestStatusPending {
		return domain.GameSession{}, domain.ErrFinishRequestNotPending
	}

	// Look up the owner's player record for the decided_by_player_id foreign key
	ownerPlayer, err := s.q.GetActivePlayerByUser(ctx, gameSessionID, userID)
	if err != nil {
		return domain.GameSession{}, err
	}

	now := s.now()
	deciderID := ownerPlayer.ID

	// Approve the request and finish the game in a transaction
	shareToken, err := GenerateShareToken()
	if err != nil {
		return domain.GameSession{}, err
	}
	err = s.store.InTx(ctx, func(q *sqlite.Queries) error {
		if err := q.UpdateFinishRequestStatus(ctx, requestID, domain.FinishRequestStatusApproved, &now, &deciderID); err != nil {
			return err
		}
		_, err = q.FinishGameSessionWithSettleAndToken(ctx, gameSessionID, shareToken, now)
		return err
	})
	if err != nil {
		return domain.GameSession{}, err
	}

	// Reload the session to return the final state
	session, err = s.q.GetGameSession(ctx, gameSessionID)
	if err != nil {
		return domain.GameSession{}, err
	}

	if s.hub != nil {
		s.hub.BroadcastToGame(ctx, gameSessionID, realtime.Event{Type: realtime.EventGameFinished, GameSessionID: gameSessionID, Version: session.Version, SentAt: now})
	}
	return session, nil
}

// RejectFinishRequest allows the game owner to reject a pending finish request.
func (s *Service) RejectFinishRequest(ctx context.Context, userID, gameSessionID, requestID string) (domain.FinishRequest, error) {
	if err := s.game.RequireMember(ctx, userID, gameSessionID); err != nil {
		return domain.FinishRequest{}, err
	}
	session, err := s.q.GetGameSession(ctx, gameSessionID)
	if err != nil {
		return domain.FinishRequest{}, err
	}
	if session.Status != domain.GameSessionStatusActive {
		return domain.FinishRequest{}, domain.ErrGameSessionFinished
	}
	if session.OwnerUserID != userID {
		return domain.FinishRequest{}, domain.ErrNotOwner
	}

	req, err := s.q.GetFinishRequest(ctx, requestID)
	if err != nil {
		return domain.FinishRequest{}, err
	}
	if req.GameSessionID != gameSessionID {
		return domain.FinishRequest{}, domain.ErrNotFound
	}
	if req.Status != domain.FinishRequestStatusPending {
		return domain.FinishRequest{}, domain.ErrFinishRequestNotPending
	}

	// Look up the owner's player record for the decided_by_player_id foreign key
	ownerPlayer, err := s.q.GetActivePlayerByUser(ctx, gameSessionID, userID)
	if err != nil {
		return domain.FinishRequest{}, err
	}

	now := s.now()
	deciderID := ownerPlayer.ID
	err = s.q.UpdateFinishRequestStatus(ctx, requestID, domain.FinishRequestStatusRejected, &now, &deciderID)
	if err != nil {
		return domain.FinishRequest{}, err
	}

	req.Status = domain.FinishRequestStatusRejected
	req.DecidedAt = &now
	req.DecidedByPlayerID = &deciderID

	if s.hub != nil {
		s.hub.BroadcastToGame(ctx, gameSessionID, realtime.Event{Type: realtime.EventFinishRequestRejected, GameSessionID: gameSessionID, Version: session.Version, SentAt: now})
	}
	return req, nil
}
