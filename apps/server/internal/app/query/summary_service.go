package query

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/xuanye/one-round/apps/server/internal/api/dto"
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
	UserID       *string `json:"userId,omitempty"`
	DisplayName  string  `json:"displayName"`
	TotalScore   int     `json:"totalScore"`
	AverageScore float64 `json:"averageScore"`
}

type FinishRequestView struct {
	ID                  string    `json:"id"`
	RequestedByPlayerID string    `json:"requestedByPlayerId"`
	RequestedByName     string    `json:"requestedByName"`
	CreatedAt           time.Time `json:"createdAt"`
}

type Summary struct {
	ID                   string             `json:"id"`
	Name                 string             `json:"name"`
	OwnerUserID          string             `json:"ownerUserId"`
	Status               string             `json:"status"`
	ScoreTransferCnt     int                `json:"scoreTransferCount"`
	Players              []PlayerSummary    `json:"players"`
	UpdatedAt            time.Time          `json:"updatedAt"`
	Version              int64              `json:"version"`
	PendingFinishRequest *FinishRequestView `json:"pendingFinishRequest,omitempty"`
	PublicShareToken     *string            `json:"publicShareToken,omitempty"`
}

type RankingItem struct {
	Rank             int     `json:"rank"`
	PlayerID         string  `json:"playerId"`
	DisplayName      string  `json:"displayName"`
	TotalScore       int     `json:"totalScore"`
	ScoreTransferCnt int     `json:"scoreTransferCount"`
	AverageScore     float64 `json:"averageScore"`
}

type ScoreTransferView struct {
	ID           string    `json:"id"`
	SequenceNo   int       `json:"sequenceNo"`
	FromPlayerID string    `json:"fromPlayerId"`
	ReceiverIDs  []string  `json:"receiverPlayerIds"`
	Amount       int       `json:"amount"`
	CreatedAt    time.Time `json:"createdAt"`
	Text         string    `json:"text"`
}

func NewService(q *sqlite.Queries, game *gamesvc.Service) *Service {
	return &Service{q: q, game: game}
}

func (s *Service) ActiveParticipants(ctx context.Context, userID, gameSessionID string) ([]domain.Player, error) {
	if err := s.game.RequireMember(ctx, userID, gameSessionID); err != nil {
		return nil, err
	}
	return s.q.ListActivePlayers(ctx, gameSessionID)
}

// MyParticipant returns the active player record for the given user in the game.
func (s *Service) MyParticipant(ctx context.Context, userID, gameSessionID string) (domain.Player, error) {
	if err := s.game.RequireMember(ctx, userID, gameSessionID); err != nil {
		return domain.Player{}, err
	}
	return s.q.GetActivePlayerByUser(ctx, gameSessionID, userID)
}

func (s *Service) Summary(ctx context.Context, userID, gameSessionID string) (Summary, error) {
	if err := s.game.RequireMember(ctx, userID, gameSessionID); err != nil {
		return Summary{}, err
	}
	g, err := s.q.GetGameSession(ctx, gameSessionID)
	if err != nil {
		return Summary{}, err
	}
	players, err := s.q.ListActivePlayers(ctx, gameSessionID)
	if err != nil {
		return Summary{}, err
	}
	summary := Summary{
		ID:               g.ID,
		Name:             g.Name,
		OwnerUserID:      g.OwnerUserID,
		Status:           string(g.Status),
		ScoreTransferCnt: g.ScoreTransferCnt,
		UpdatedAt:        g.UpdatedAt,
		Version:          g.Version,
		PublicShareToken: g.PublicShareToken,
	}
	for _, p := range players {
		summary.Players = append(summary.Players, PlayerSummary{
			ID:           p.ID,
			UserID:       p.UserID,
			DisplayName:  p.DisplayName,
			TotalScore:   p.TotalScore,
			AverageScore: average(p.TotalScore, g.ScoreTransferCnt),
		})
	}

	pending, err := s.q.GetPendingFinishRequest(ctx, gameSessionID)
	if err != nil {
		return Summary{}, err
	}
	if pending != nil {
		var requesterName string
		for _, p := range players {
			if p.ID == pending.RequestedByPlayerID {
				requesterName = p.DisplayName
				break
			}
		}
		if requesterName == "" {
			reqPlayer, err := s.q.GetPlayer(ctx, gameSessionID, pending.RequestedByPlayerID)
			if err == nil {
				requesterName = reqPlayer.DisplayName
			} else {
				requesterName = "未知玩家"
			}
		}
		summary.PendingFinishRequest = &FinishRequestView{
			ID:                  pending.ID,
			RequestedByPlayerID: pending.RequestedByPlayerID,
			RequestedByName:     requesterName,
			CreatedAt:           pending.CreatedAt,
		}
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

func average(total, roundCount int) float64 {
	if roundCount == 0 {
		return 0
	}
	return math.Round(float64(total)/float64(roundCount)*100) / 100
}

func (s *Service) ListScoreTransfers(ctx context.Context, userID, gameSessionID string, beforeSequenceNo *int, limit int) ([]ScoreTransferView, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if err := s.game.RequireMember(ctx, userID, gameSessionID); err != nil {
		return nil, err
	}
	transfers, err := s.q.ListScoreTransfersPaginated(ctx, gameSessionID, beforeSequenceNo, limit)
	if err != nil {
		return nil, err
	}

	// Build a player display name lookup for formatting
	players, err := s.q.ListHistoricalPlayers(ctx, gameSessionID)
	if err != nil {
		return nil, err
	}
	nameMap := make(map[string]string, len(players))
	for _, p := range players {
		nameMap[p.ID] = p.DisplayName
	}

	views := make([]ScoreTransferView, 0, len(transfers))
	for _, t := range transfers {
		receiverIDs := make([]string, 0, len(t.Receivers))
		receiverNames := make([]string, 0, len(t.Receivers))
		for _, r := range t.Receivers {
			receiverIDs = append(receiverIDs, r.PlayerID)
			if n, ok := nameMap[r.PlayerID]; ok {
				receiverNames = append(receiverNames, n)
			} else {
				receiverNames = append(receiverNames, r.PlayerID)
			}
		}
		fromName := nameMap[t.FromPlayerID]
		if fromName == "" {
			fromName = t.FromPlayerID
		}
		text := formatTransferText(fromName, receiverNames, t.Amount)
		views = append(views, ScoreTransferView{
			ID:           t.ID,
			SequenceNo:   t.SequenceNo,
			FromPlayerID: t.FromPlayerID,
			ReceiverIDs:  receiverIDs,
			Amount:       t.Amount,
			CreatedAt:    t.CreatedAt,
			Text:         text,
		})
	}
	return views, nil
}

func formatTransferText(from string, receivers []string, amount int) string {
	if len(receivers) == 1 {
		return fmt.Sprintf("%s 给 %s +%d", from, receivers[0], amount)
	}
	return fmt.Sprintf("%s 给 %s 各 +%d", from, strings.Join(receivers, "、"), amount)
}

// History returns a page of settled game sessions for the given user.
// Only finished, non-voided games where the user has a historical player record are listed.
func (s *Service) History(ctx context.Context, userID string, beforeSettledAt *time.Time, limit int) (dto.HistoryPage, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	sessions, err := s.q.ListSettledGamesForUser(ctx, userID, beforeSettledAt, limit)
	if err != nil {
		return dto.HistoryPage{}, err
	}

	items := make([]dto.HistoryItem, 0, len(sessions))
	for _, g := range sessions {
		// Get the user's player record to compute myFinalScore
		player, err := s.q.GetHistoricalPlayerByUser(ctx, g.ID, userID)
		if err != nil {
			return dto.HistoryPage{}, err
		}

		item := dto.HistoryItem{
			ID:                 g.ID,
			Name:               g.Name,
			ScoreTransferCount: g.ScoreTransferCnt,
			MyFinalScore:       player.TotalScore,
		}
		if g.SettledAt != nil {
			item.SettledAt = *g.SettledAt
		}
		items = append(items, item)
	}

	var nextCursor *string
	if len(sessions) == limit && len(sessions) > 0 {
		last := sessions[len(sessions)-1]
		if last.SettledAt != nil {
			c := last.SettledAt.UTC().Format(time.RFC3339)
			nextCursor = &c
		}
	}

	return dto.HistoryPage{Items: items, NextCursor: nextCursor}, nil
}

// SettlementDetail returns settlement details for a finished game session.
// The user must have been a historical participant in the game.
// Participants include active and inactive historical participants, sorted by final score desc, joined order asc.
// Score transfers are paginated desc.
func (s *Service) SettlementDetail(ctx context.Context, userID, gameSessionID string, beforeSequenceNo *int, limit int) (dto.SettlementDetail, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	// Verify the user has a historical player record in this game
	_, err := s.q.GetHistoricalPlayerByUser(ctx, gameSessionID, userID)
	if err != nil {
		return dto.SettlementDetail{}, err
	}

	g, err := s.q.GetGameSession(ctx, gameSessionID)
	if err != nil {
		return dto.SettlementDetail{}, err
	}
	if g.Status != domain.GameSessionStatusFinished {
		return dto.SettlementDetail{}, domain.ErrGameSessionFinished
	}

	// List all historical players sorted by final score desc, joined order asc
	players, err := s.q.ListHistoricalPlayersForUser(ctx, gameSessionID, userID)
	if err != nil {
		return dto.SettlementDetail{}, err
	}

	participants := make([]dto.SettlementParticipant, 0, len(players))
	for _, p := range players {
		participants = append(participants, dto.SettlementParticipant{
			ID:          p.ID,
			DisplayName: p.DisplayName,
			FinalScore:  p.TotalScore,
		})
	}

	// Get paginated score transfers
	transfers, err := s.q.ListScoreTransfersPaginated(ctx, gameSessionID, beforeSequenceNo, limit)
	if err != nil {
		return dto.SettlementDetail{}, err
	}

	transferSummaries := make([]dto.ScoreTransferSummary, 0, len(transfers))
	for _, t := range transfers {
		transferSummaries = append(transferSummaries, dto.ScoreTransferSummary{
			ID:        t.ID,
			Amount:    t.Amount,
			CreatedAt: t.CreatedAt.UTC().Format(time.RFC3339),
		})
	}

	var nextCursor *int
	if len(transfers) == limit && len(transfers) > 0 {
		c := transfers[len(transfers)-1].SequenceNo
		nextCursor = &c
	}

	settledAt := time.Time{}
	if g.SettledAt != nil {
		settledAt = *g.SettledAt
	}

	return dto.SettlementDetail{
		ID:               g.ID,
		Name:             g.Name,
		SettledAt:        settledAt,
		Participants:     participants,
		ScoreTransfers:   transferSummaries,
		NextCursor:       nextCursor,
		PublicShareToken: g.PublicShareToken,
	}, nil
}

// PublicSettlement returns public settlement details for a finished game session.
// It displays names, settlement date, display names, final scores, but no avatars and no score transfer details.
func (s *Service) PublicSettlement(ctx context.Context, shareToken string) (dto.PublicSettlement, error) {
	g, err := s.q.GetGameSessionByPublicShareToken(ctx, shareToken)
	if err != nil {
		return dto.PublicSettlement{}, err
	}

	// List all historical players sorted by final score desc, joined order asc
	players, err := s.q.ListHistoricalPlayers(ctx, g.ID)
	if err != nil {
		return dto.PublicSettlement{}, err
	}

	participants := make([]dto.SettlementParticipant, 0, len(players))
	for _, p := range players {
		participants = append(participants, dto.SettlementParticipant{
			ID:          p.ID,
			DisplayName: p.DisplayName,
			AvatarURL:   nil, // Public share omits avatars
			FinalScore:  p.TotalScore,
		})
	}

	settledAt := time.Time{}
	if g.SettledAt != nil {
		settledAt = *g.SettledAt
	}

	return dto.PublicSettlement{
		GameSessionID:  g.ID,
		Name:           g.Name,
		SettledAt:      settledAt,
		Participants:   participants,
		ScoreTransfers: nil, // Public share omits transfer details
	}, nil
}

func (s *Service) UserStats(ctx context.Context, userID string) (dto.UserStatsResponse, error) {
	count, maxScore, err := s.q.GetUserStats(ctx, userID)
	if err != nil {
		return dto.UserStatsResponse{}, err
	}
	return dto.UserStatsResponse{
		TotalGames: count,
		MaxScore:   maxScore,
	}, nil
}

var _ = domain.GameSession{}

