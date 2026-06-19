package query

import (
	"context"
	"math"
	"time"

	gamesvc "github.com/xuanye/one-round/apps/server/internal/app/game"
	"github.com/xuanye/one-round/apps/server/internal/domain"
	"github.com/xuanye/one-round/apps/server/internal/infra/sqlite"
)

type Service struct {
	q    *sqlite.Queries
	game *gamesvc.Service
}

type PlayerSummary struct {
	ID           string  `json:"id"`
	DisplayName  string  `json:"displayName"`
	TotalScore   int     `json:"totalScore"`
	AverageScore float64 `json:"averageScore"`
}

type RoundScoreView struct {
	PlayerID string `json:"playerId"`
	Score    int    `json:"score"`
}

type RecentRound struct {
	ID        string           `json:"id"`
	RoundNo   int              `json:"roundNo"`
	CreatedAt time.Time        `json:"createdAt"`
	Scores    []RoundScoreView `json:"scores"`
}

type Summary struct {
	ID               string          `json:"id"`
	Name             string          `json:"name"`
	Status           string          `json:"status"`
	ScoreTransferCnt int             `json:"scoreTransferCount"`
	Players          []PlayerSummary `json:"players"`
	RecentRounds     []RecentRound   `json:"recentRounds"`
	UpdatedAt        time.Time       `json:"updatedAt"`
	Version          int64           `json:"version"`
}

type RankingItem struct {
	Rank             int     `json:"rank"`
	PlayerID         string  `json:"playerId"`
	DisplayName      string  `json:"displayName"`
	TotalScore       int     `json:"totalScore"`
	ScoreTransferCnt int     `json:"scoreTransferCount"`
	AverageScore     float64 `json:"averageScore"`
}

func NewService(q *sqlite.Queries, game *gamesvc.Service) *Service {
	return &Service{q: q, game: game}
}

func (s *Service) Summary(ctx context.Context, userID, gameSessionID string) (Summary, error) {
	if err := s.game.RequireMember(ctx, userID, gameSessionID); err != nil {
		return Summary{}, err
	}
	g, err := s.q.GetGameSession(ctx, gameSessionID)
	if err != nil {
		return Summary{}, err
	}
	players, err := s.q.ListRanking(ctx, gameSessionID)
	if err != nil {
		return Summary{}, err
	}
	rounds, err := s.RecentRounds(ctx, userID, gameSessionID, 20)
	if err != nil {
		return Summary{}, err
	}
	summary := Summary{ID: g.ID, Name: g.Name, Status: string(g.Status), ScoreTransferCnt: g.ScoreTransferCnt, RecentRounds: rounds, UpdatedAt: g.UpdatedAt, Version: g.Version}
	for _, p := range players {
		summary.Players = append(summary.Players, PlayerSummary{ID: p.ID, DisplayName: p.DisplayName, TotalScore: p.TotalScore, AverageScore: average(p.TotalScore, g.ScoreTransferCnt)})
	}
	return summary, nil
}

func (s *Service) Ranking(ctx context.Context, userID, gameSessionID string) ([]RankingItem, error) {
	if err := s.game.RequireMember(ctx, userID, gameSessionID); err != nil {
		return nil, err
	}
	g, err := s.q.GetGameSession(ctx, gameSessionID)
	if err != nil {
		return nil, err
	}
	players, err := s.q.ListRanking(ctx, gameSessionID)
	if err != nil {
		return nil, err
	}
	items := make([]RankingItem, 0, len(players))
	for i, p := range players {
		items = append(items, RankingItem{Rank: i + 1, PlayerID: p.ID, DisplayName: p.DisplayName, TotalScore: p.TotalScore, ScoreTransferCnt: g.ScoreTransferCnt, AverageScore: average(p.TotalScore, g.ScoreTransferCnt)})
	}
	return items, nil
}

func (s *Service) RecentRounds(ctx context.Context, userID, gameSessionID string, limit int) ([]RecentRound, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if err := s.game.RequireMember(ctx, userID, gameSessionID); err != nil {
		return nil, err
	}
	rounds, err := s.q.ListRecentRounds(ctx, gameSessionID, limit)
	if err != nil {
		return nil, err
	}
	out := make([]RecentRound, 0, len(rounds))
	for _, r := range rounds {
		item := RecentRound{ID: r.ID, RoundNo: r.RoundNo, CreatedAt: r.CreatedAt}
		for _, score := range r.Scores {
			item.Scores = append(item.Scores, RoundScoreView{PlayerID: score.PlayerID, Score: score.Score})
		}
		out = append(out, item)
	}
	return out, nil
}

func average(total, roundCount int) float64 {
	if roundCount == 0 {
		return 0
	}
	return math.Round(float64(total)/float64(roundCount)*100) / 100
}

var _ = domain.GameSession{}
