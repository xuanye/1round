package settlement

import (
	"context"
	"crypto/rand"
	"encoding/base64"
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
