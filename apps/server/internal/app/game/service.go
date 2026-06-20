package game

import (
	"context"
	"crypto/rand"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/xuanye/one-round/apps/server/internal/domain"
	"github.com/xuanye/one-round/apps/server/internal/infra/sqlite"
	"github.com/xuanye/one-round/apps/server/internal/realtime"
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

func (s *Service) Create(ctx context.Context, userID, name string, maxParticipants *int) (domain.GameSession, error) {
	if strings.TrimSpace(name) == "" {
		return domain.GameSession{}, domain.ErrInvalidArgument
	}
	if maxParticipants != nil && (*maxParticipants < 2 || *maxParticipants > 10) {
		return domain.GameSession{}, domain.ErrInvalidArgument
	}
	current, err := s.q.GetCurrentGameForUser(ctx, userID)
	if err != nil {
		return domain.GameSession{}, err
	}
	if current != nil {
		return domain.GameSession{}, domain.ErrActiveGameExists
	}
	now := s.now()
	code, err := GenerateInviteCode()
	if err != nil {
		return domain.GameSession{}, err
	}
	session := domain.GameSession{
		ID: uuid.NewString(), Name: strings.TrimSpace(name), InviteCode: code, OwnerUserID: userID,
		Status: domain.GameSessionStatusActive, MaxParticipants: maxParticipants, ScoreTransferCnt: 0, Version: 1,
		CreatedAt: now, UpdatedAt: now,
	}
	member := domain.GameMember{ID: uuid.NewString(), GameSessionID: session.ID, UserID: userID, Role: domain.GameMemberRoleOwner, JoinedAt: now}
	displayName := defaultDisplayName(userID)
	user, err := s.q.GetUserByID(ctx, userID)
	if err == nil && user.DisplayName != nil && strings.TrimSpace(*user.DisplayName) != "" {
		displayName = strings.TrimSpace(*user.DisplayName)
	}
	ownerPlayer := domain.Player{
		ID: uuid.NewString(), GameSessionID: session.ID, UserID: &userID, DisplayName: displayName,
		Active: true, JoinedOrder: 1, TotalScore: 0, CreatedAt: now, UpdatedAt: now,
	}
	err = s.store.InTx(ctx, func(q *sqlite.Queries) error {
		if err := q.CreateGameSession(ctx, session, member); err != nil {
			return err
		}
		return q.CreatePlayer(ctx, ownerPlayer)
	})
	return session, err
}

func (s *Service) Current(ctx context.Context, userID string) (*domain.GameSession, error) {
	return s.q.GetCurrentGameForUser(ctx, userID)
}

type JoinPreview struct {
	GameSessionID          string          `json:"gameSessionId"`
	Name                   string          `json:"name"`
	OwnerDisplayName       string          `json:"ownerDisplayName"`
	ParticipantCount       int             `json:"participantCount"`
	MaxParticipants        *int            `json:"maxParticipants"`
	Participants           []PlayerPreview `json:"participants"`
	CurrentUserDisplayName string          `json:"currentUserDisplayName"`
	AlreadyJoined          bool            `json:"alreadyJoined"`
}

type PlayerPreview struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
}

func defaultDisplayName(userID string) string {
	// Derive a 2-digit numeric suffix from the userID bytes so names look like "老书记42"
	sum := 0
	for i := 0; i < len(userID); i++ {
		sum += int(userID[i])
	}
	return fmt.Sprintf("老书记%02d", sum%100)
}

func (s *Service) JoinPreview(ctx context.Context, userID, inviteCode string) (JoinPreview, error) {
	session, err := s.q.GetGameSessionByInviteCode(ctx, strings.ToUpper(strings.TrimSpace(inviteCode)))
	if err != nil {
		return JoinPreview{}, err
	}
	if session.Status == domain.GameSessionStatusFinished || session.Status == domain.GameSessionStatusVoided {
		return JoinPreview{}, domain.ErrGameSessionFinished
	}

	// Use active participants for count and display
	activePlayers, err := s.q.ListActivePlayers(ctx, session.ID)
	if err != nil {
		return JoinPreview{}, err
	}

	preview := JoinPreview{
		GameSessionID:    session.ID,
		Name:             session.Name,
		MaxParticipants:  session.MaxParticipants,
		ParticipantCount: len(activePlayers),
	}

	var ownerDisplayName string
	alreadyJoined := false
	for _, p := range activePlayers {
		preview.Participants = append(preview.Participants, PlayerPreview{ID: p.ID, DisplayName: p.DisplayName})
		if p.UserID != nil && *p.UserID == session.OwnerUserID {
			ownerDisplayName = p.DisplayName
		}
		if p.UserID != nil && *p.UserID == userID {
			alreadyJoined = true
		}
	}
	preview.OwnerDisplayName = ownerDisplayName
	preview.AlreadyJoined = alreadyJoined

	// Check historical players for current user's display name (for rejoin scenario)
	var currentUserDisplayName string
	historicalPlayers, err := s.q.ListHistoricalPlayers(ctx, session.ID)
	if err != nil {
		return JoinPreview{}, err
	}
	for _, p := range historicalPlayers {
		if p.UserID != nil && *p.UserID == userID {
			currentUserDisplayName = p.DisplayName
			break
		}
	}

	// If user is not yet a participant, prefer the latest global nickname.
	if currentUserDisplayName == "" {
		user, err := s.q.GetUserByID(ctx, userID)
		if err != nil {
			return JoinPreview{}, err
		}
		if user.DisplayName != nil && strings.TrimSpace(*user.DisplayName) != "" {
			currentUserDisplayName = strings.TrimSpace(*user.DisplayName)
		} else {
			currentUserDisplayName = defaultDisplayName(userID)
		}
	}
	preview.CurrentUserDisplayName = currentUserDisplayName

	return preview, nil
}

func (s *Service) Join(ctx context.Context, userID, inviteCode, displayName string) (string, error) {
	session, err := s.q.GetGameSessionByInviteCode(ctx, strings.ToUpper(strings.TrimSpace(inviteCode)))
	if err != nil {
		return "", err
	}
	if session.Status == domain.GameSessionStatusFinished || session.Status == domain.GameSessionStatusVoided {
		return "", domain.ErrGameSessionFinished
	}

	// Check if user is already an active participant in this game (early return, no displayName needed)
	if _, err := s.q.GetActivePlayerByUser(ctx, session.ID, userID); err == nil {
		return session.ID, nil
	} else if err != domain.ErrNotFound {
		return "", err
	}

	if strings.TrimSpace(displayName) == "" {
		return "", domain.ErrInvalidArgument
	}

	// Check if user has another current game (different from this one)
	current, err := s.q.GetCurrentGameForUser(ctx, userID)
	if err != nil {
		return "", err
	}
	if current != nil && current.ID != session.ID {
		return "", domain.ErrActiveGameExists
	}

	displayName = strings.TrimSpace(displayName)

	// Check if user was a historical (inactive) participant -- reactivate
	historicalPlayer, err := s.q.GetHistoricalPlayerByUser(ctx, session.ID, userID)
	if err != nil && err != domain.ErrNotFound {
		return "", err
	}
	if err == nil {
		// Reactivate: name unique check excluding self
		if err := s.checkDisplayNameUnique(ctx, session.ID, displayName, historicalPlayer.ID); err != nil {
			return "", err
		}
		now := s.now()
		err = s.store.InTx(ctx, func(q *sqlite.Queries) error {
			// Re-check game status inside transaction to prevent joins after settlement/void
			currentSession, err := q.GetGameSession(ctx, session.ID)
			if err != nil {
				return err
			}
			if currentSession.Status != domain.GameSessionStatusActive {
				return domain.ErrGameSessionFinished
			}
			// Re-check capacity inside transaction to prevent concurrent joins exceeding limit
			if session.MaxParticipants != nil {
				activeCount, err := q.CountActiveParticipants(ctx, session.ID)
				if err != nil {
					return err
				}
				if activeCount >= *session.MaxParticipants {
					return domain.ErrGameCapacityFull
				}
			}
			if err := q.ReactivatePlayer(ctx, historicalPlayer.ID, session.ID, displayName, 0, now); err != nil {
				return err
			}
			// Only add membership if not already a member (user may have left without being removed from game_members)
			isMember, err := q.IsGameMember(ctx, session.ID, userID)
			if err != nil {
				return err
			}
			if !isMember {
				return q.AddGameMember(ctx, domain.GameMember{ID: uuid.NewString(), GameSessionID: session.ID, UserID: userID, Role: domain.GameMemberRoleMember, JoinedAt: now})
			}
			return nil
		})
		if err != nil {
			return "", err
		}
		if s.hub != nil {
			s.hub.BroadcastToGame(ctx, session.ID, realtime.Event{Type: realtime.EventParticipantJoined, GameSessionID: session.ID, Version: session.Version, SentAt: s.now()})
		}
		return session.ID, nil
	}

	// Brand new participant
	if err := s.checkDisplayNameUnique(ctx, session.ID, displayName, ""); err != nil {
		return "", err
	}
	now := s.now()
	err = s.store.InTx(ctx, func(q *sqlite.Queries) error {
		// Re-check game status inside transaction to prevent joins after settlement/void
		currentSession, err := q.GetGameSession(ctx, session.ID)
		if err != nil {
			return err
		}
		if currentSession.Status != domain.GameSessionStatusActive {
			return domain.ErrGameSessionFinished
		}
		// Re-check capacity inside transaction to prevent concurrent joins exceeding limit
		if session.MaxParticipants != nil {
			activeCount, err := q.CountActiveParticipants(ctx, session.ID)
			if err != nil {
				return err
			}
			if activeCount >= *session.MaxParticipants {
				return domain.ErrGameCapacityFull
			}
		}
		joinedOrder, err := q.NextJoinedOrder(ctx, session.ID)
		if err != nil {
			return err
		}
		p := domain.Player{
			ID: uuid.NewString(), GameSessionID: session.ID, UserID: &userID, DisplayName: displayName,
			Active: true, JoinedOrder: joinedOrder, TotalScore: 0, CreatedAt: now, UpdatedAt: now,
		}
		if err := q.CreatePlayer(ctx, p); err != nil {
			return err
		}
		return q.AddGameMember(ctx, domain.GameMember{ID: uuid.NewString(), GameSessionID: session.ID, UserID: userID, Role: domain.GameMemberRoleMember, JoinedAt: now})
	})
	if err != nil {
		return "", err
	}
	if s.hub != nil {
		s.hub.BroadcastToGame(ctx, session.ID, realtime.Event{Type: realtime.EventParticipantJoined, GameSessionID: session.ID, Version: session.Version, SentAt: s.now()})
	}
	return session.ID, nil
}

func (s *Service) checkDisplayNameUnique(ctx context.Context, gameSessionID, displayName, excludePlayerID string) error {
	players, err := s.q.ListHistoricalPlayers(ctx, gameSessionID)
	if err != nil {
		return err
	}
	for _, p := range players {
		if p.ID == excludePlayerID {
			continue
		}
		if p.DisplayName == displayName {
			return domain.ErrDuplicateDisplayName
		}
	}
	return nil
}

func (s *Service) GetForMember(ctx context.Context, userID, gameSessionID string) (domain.GameSession, error) {
	if err := s.requireMember(ctx, userID, gameSessionID); err != nil {
		return domain.GameSession{}, err
	}
	return s.q.GetGameSession(ctx, gameSessionID)
}

func (s *Service) GetForHistoricalMember(ctx context.Context, userID, gameSessionID string) (domain.GameSession, error) {
	ok, err := s.q.IsGameMember(ctx, gameSessionID, userID)
	if err != nil {
		return domain.GameSession{}, err
	}
	if !ok {
		return domain.GameSession{}, domain.ErrGameMemberRequired
	}
	return s.q.GetGameSession(ctx, gameSessionID)
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
