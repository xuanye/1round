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
	q    *sqlite.Queries
	game *gamesvc.Service
	hub  realtime.Hub
	now  func() time.Time
}

func NewService(q *sqlite.Queries, game *gamesvc.Service, hub realtime.Hub, now func() time.Time) *Service {
	return &Service{q: q, game: game, hub: hub, now: now}
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
	p := domain.Player{ID: uuid.NewString(), GameSessionID: gameSessionID, DisplayName: strings.TrimSpace(displayName), CreatedAt: now, UpdatedAt: now}
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
