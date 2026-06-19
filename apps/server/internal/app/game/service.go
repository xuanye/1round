package game

import (
	"context"
	"crypto/rand"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redhu/one-round/apps/server/internal/domain"
	"github.com/redhu/one-round/apps/server/internal/infra/sqlite"
	"github.com/redhu/one-round/apps/server/internal/realtime"
)

const inviteAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

type Service struct {
	store *sqlite.Store
	q     *sqlite.Queries
	hub   realtime.Hub
	now   func() time.Time
}

func NewService(store *sqlite.Store, q *sqlite.Queries, hub realtime.Hub, now func() time.Time) *Service {
	return &Service{store: store, q: q, hub: hub, now: now}
}

func GenerateInviteCode() (string, error) {
	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	for i := range buf {
		buf[i] = inviteAlphabet[int(buf[i])%len(inviteAlphabet)]
	}
	return string(buf), nil
}

func ValidInviteCode(code string) bool {
	if len(code) != 6 {
		return false
	}
	for _, r := range code {
		if !strings.ContainsRune(inviteAlphabet, r) {
			return false
		}
	}
	return true
}

func (s *Service) Create(ctx context.Context, userID, name string, zeroSumRequired bool) (domain.GameSession, error) {
	if strings.TrimSpace(name) == "" {
		return domain.GameSession{}, domain.ErrInvalidArgument
	}
	now := s.now()
	code, err := GenerateInviteCode()
	if err != nil {
		return domain.GameSession{}, err
	}
	session := domain.GameSession{
		ID: uuid.NewString(), Name: strings.TrimSpace(name), InviteCode: code, OwnerUserID: userID,
		Status: domain.GameSessionStatusActive, ZeroSumRequired: zeroSumRequired, RoundCount: 0, Version: 1,
		CreatedAt: now, UpdatedAt: now,
	}
	member := domain.GameMember{ID: uuid.NewString(), GameSessionID: session.ID, UserID: userID, Role: domain.GameMemberRoleOwner, JoinedAt: now}
	err = s.store.InTx(ctx, func(q *sqlite.Queries) error {
		return q.CreateGameSession(ctx, session, member)
	})
	return session, err
}

func (s *Service) Join(ctx context.Context, userID, inviteCode string) (string, error) {
	session, err := s.q.GetGameSessionByInviteCode(ctx, strings.ToUpper(strings.TrimSpace(inviteCode)))
	if err != nil {
		return "", err
	}
	if session.Status == domain.GameSessionStatusFinished {
		return "", domain.ErrGameSessionFinished
	}
	err = s.q.AddGameMember(ctx, domain.GameMember{ID: uuid.NewString(), GameSessionID: session.ID, UserID: userID, Role: domain.GameMemberRoleMember, JoinedAt: s.now()})
	return session.ID, err
}

func (s *Service) GetForMember(ctx context.Context, userID, gameSessionID string) (domain.GameSession, error) {
	if err := s.requireMember(ctx, userID, gameSessionID); err != nil {
		return domain.GameSession{}, err
	}
	return s.q.GetGameSession(ctx, gameSessionID)
}

func (s *Service) Finish(ctx context.Context, userID, gameSessionID string) (domain.GameSession, error) {
	if err := s.requireMember(ctx, userID, gameSessionID); err != nil {
		return domain.GameSession{}, err
	}
	session, err := s.q.FinishGameSession(ctx, gameSessionID, s.now())
	if err == nil && s.hub != nil {
		s.hub.BroadcastToGame(ctx, gameSessionID, realtime.Event{Type: realtime.EventGameFinished, GameSessionID: gameSessionID, Version: session.Version, SentAt: s.now()})
	}
	return session, err
}

func (s *Service) RequireMember(ctx context.Context, userID, gameSessionID string) error {
	return s.requireMember(ctx, userID, gameSessionID)
}

func (s *Service) requireMember(ctx context.Context, userID, gameSessionID string) error {
	ok, err := s.q.IsGameMember(ctx, gameSessionID, userID)
	if err != nil {
		return err
	}
	if !ok {
		return domain.ErrGameMemberRequired
	}
	return nil
}
