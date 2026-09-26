package query

import (
	"context"
	"sort"
	"time"

	"github.com/xuanye/one-round/apps/server/internal/api/dto"
	"github.com/xuanye/one-round/apps/server/internal/domain"
)

func (s *Service) Performance(ctx context.Context, userID string, start, end time.Time) (dto.Performance, error) {
	result := dto.Performance{Trend: []dto.PerformancePoint{}, Players: []dto.PerformancePlayer{}, RecentGames: []dto.PerformanceGame{}}
	if userID == "" {
		return result, domain.ErrUnauthorized
	}
	if !start.Before(end) || end.Sub(start) > 367*24*time.Hour {
		return result, domain.ErrInvalidArgument
	}
	rows, err := s.q.ListPerformance(ctx, userID, start, end)
	if err != nil {
		return result, err
	}
	players := map[string]dto.PerformancePlayer{}
	for i := 0; i < len(rows); {
		j := i
		game := dto.PerformanceGame{ID: rows[i].GameID, Name: rows[i].GameName, SettledAt: rows[i].SettledAt}
		best := rows[i].Score
		for j < len(rows) && rows[j].GameID == game.ID {
			row := rows[j]
			if row.Score > best {
				best = row.Score
			}
			if row.UserID == userID {
				game.MyFinalScore = row.Score
			}
			id := row.UserID
			if id == "" {
				id = row.PlayerID
			}
			player := players[id]
			player.ID, player.DisplayName, player.AvatarURL = id, row.DisplayName, row.AvatarURL
			player.IsMe = row.UserID == userID
			player.TotalScore += row.Score
			players[id] = player
			j++
		}
		game.ParticipantCount = j - i
		// A win means a positive final score tied for the highest in the game.
		if game.MyFinalScore > 0 && game.MyFinalScore == best {
			result.Wins++
		}
		if result.TotalGames == 0 || game.MyFinalScore > result.MaxScore {
			result.MaxScore = game.MyFinalScore
		}
		result.TotalGames++
		result.TotalScore += game.MyFinalScore
		result.Trend = append(result.Trend, dto.PerformancePoint{SettledAt: game.SettledAt, Score: result.TotalScore})
		result.RecentGames = append(result.RecentGames, game)
		i = j
	}
	for _, player := range players {
		result.Players = append(result.Players, player)
	}
	sort.Slice(result.Players, func(i, j int) bool {
		if result.Players[i].TotalScore != result.Players[j].TotalScore {
			return result.Players[i].TotalScore > result.Players[j].TotalScore
		}
		return result.Players[i].ID < result.Players[j].ID
	})
	recent := result.RecentGames
	result.RecentGames = []dto.PerformanceGame{}
	for i := len(recent) - 1; i >= 0 && len(result.RecentGames) < 2; i-- {
		result.RecentGames = append(result.RecentGames, recent[i])
	}
	return result, nil
}
