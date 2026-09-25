package query

import (
	"context"

	"github.com/xuanye/one-round/apps/server/internal/api/dto"
	"github.com/xuanye/one-round/apps/server/internal/domain"
)

type transferProjection struct {
	initiatedByName string
	changes         []dto.ScoreChange
}

// projectTransferHistory replays persisted transfers so paginated records keep
// their balances at the time of each transfer, including later reversals.
func (s *Service) projectTransferHistory(ctx context.Context, gameSessionID string, players []domain.Player) (map[string]transferProjection, error) {
	ledger, err := s.q.ListScoreTransferLedger(ctx, gameSessionID)
	if err != nil {
		return nil, err
	}

	nameByPlayerID := make(map[string]string, len(players))
	nameByUserID := make(map[string]string, len(players))
	for _, player := range players {
		nameByPlayerID[player.ID] = player.DisplayName
		if player.UserID != nil {
			nameByUserID[*player.UserID] = player.DisplayName
		}
	}

	balances := make(map[string]int, len(players))
	projected := make(map[string]transferProjection, len(ledger))
	for _, transfer := range ledger {
		changes := make([]dto.ScoreChange, 0, len(transfer.ReceiverIDs)+1)
		appendChange := func(playerID string, delta int) {
			before := balances[playerID]
			after := before + delta
			balances[playerID] = after
			name := nameByPlayerID[playerID]
			if name == "" {
				name = playerID
			}
			changes = append(changes, dto.ScoreChange{
				PlayerID: playerID, PlayerName: name, Before: before, After: after, Delta: delta,
			})
		}

		senderDelta := -transfer.Amount * len(transfer.ReceiverIDs)
		receiverDelta := transfer.Amount
		if transfer.Kind == domain.ScoreTransferKindReversal {
			senderDelta = -senderDelta
			receiverDelta = -receiverDelta
		}
		appendChange(transfer.FromPlayerID, senderDelta)
		for _, receiverID := range transfer.ReceiverIDs {
			appendChange(receiverID, receiverDelta)
		}
		initiator := nameByUserID[transfer.CreatedByUserID]
		if initiator == "" {
			initiator = "未知参与者"
		}
		projected[transfer.ID] = transferProjection{initiatedByName: initiator, changes: changes}
	}
	return projected, nil
}
