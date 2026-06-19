package round

import (
	"context"
	"time"

	"github.com/google/uuid"
	gamesvc "github.com/redhu/one-round/apps/server/internal/app/game"
	"github.com/redhu/one-round/apps/server/internal/domain"
	"github.com/redhu/one-round/apps/server/internal/infra/sqlite"
	"github.com/redhu/one-round/apps/server/internal/realtime"
)

type Service struct {
	store *sqlite.Store
	q     *sqlite.Queries
	game  *gamesvc.Service
	hub   realtime.Hub
	now   func() time.Time
}

type ScoreInput struct {
	PlayerID string `json:"playerId"`
	Score    int    `json:"score"`
}

type SubmitResult struct {
	RoundID string `json:"roundId"`
	RoundNo int    `json:"roundNo"`
	Version int64  `json:"version"`
}

func NewService(store *sqlite.Store, q *sqlite.Queries, game *gamesvc.Service, hub realtime.Hub, now func() time.Time) *Service {
	return &Service{store: store, q: q, game: game, hub: hub, now: now}
}

func ValidateZeroSum(scores []ScoreInput) error {
	total := 0
	for _, s := range scores {
		total += s.Score
	}
	if total != 0 {
		return domain.ErrScoreTotalMustBeZero
	}
	return nil
}

func (s *Service) Submit(ctx context.Context, userID, gameSessionID string, scores []ScoreInput, note *string) (SubmitResult, error) {
	if err := s.game.RequireMember(ctx, userID, gameSessionID); err != nil {
		return SubmitResult{}, err
	}
	session, err := s.q.GetGameSession(ctx, gameSessionID)
	if err != nil {
		return SubmitResult{}, err
	}
	if session.Status == domain.GameSessionStatusFinished {
		return SubmitResult{}, domain.ErrGameSessionFinished
	}
	players, err := s.q.ListPlayers(ctx, gameSessionID)
	if err != nil {
		return SubmitResult{}, err
	}
	if len(players) == 0 || len(scores) != len(players) {
		return SubmitResult{}, domain.ErrInvalidPlayer
	}
	scoreByPlayer := map[string]int{}
	for _, input := range scores {
		scoreByPlayer[input.PlayerID] = input.Score
	}
	for _, p := range players {
		if _, ok := scoreByPlayer[p.ID]; !ok {
			return SubmitResult{}, domain.ErrInvalidPlayer
		}
	}
	if session.ZeroSumRequired {
		if err := ValidateZeroSum(scores); err != nil {
			return SubmitResult{}, err
		}
	}
	now := s.now()
	roundID := uuid.NewString()
	roundNo := session.RoundCount + 1
	version := session.Version + 1
	r := domain.Round{ID: roundID, GameSessionID: gameSessionID, RoundNo: roundNo, CreatedByUserID: userID, Note: note, CreatedAt: now}
	for _, input := range scores {
		r.Scores = append(r.Scores, domain.RoundScore{ID: uuid.NewString(), RoundID: roundID, PlayerID: input.PlayerID, Score: input.Score})
	}
	err = s.store.InTx(ctx, func(q *sqlite.Queries) error {
		return q.InsertRoundWithScores(ctx, r, version)
	})
	if err != nil {
		return SubmitResult{}, err
	}
	if s.hub != nil {
		s.hub.BroadcastToGame(ctx, gameSessionID, realtime.Event{Type: realtime.EventRoundSubmitted, GameSessionID: gameSessionID, Version: version, Payload: map[string]any{"roundNo": roundNo}, SentAt: s.now()})
	}
	return SubmitResult{RoundID: roundID, RoundNo: roundNo, Version: version}, nil
}
