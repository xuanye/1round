package query

import (
	"testing"
)

func TestSummaryRoundStatusProjection(t *testing.T) {
	summary := Summary{
		RoundStatus: RoundStatus{
			RoundNo:            4,
			Status:             "active",
			PendingPlayerIDs:   []string{"p2"},
			PendingPlayerNames: []string{"李四"},
			CanStartNextRound:  false,
		},
	}

	if summary.RoundStatus.RoundNo != 4 {
		t.Fatalf("expected round 4, got %d", summary.RoundStatus.RoundNo)
	}
	if len(summary.RoundStatus.PendingPlayerNames) != 1 || summary.RoundStatus.PendingPlayerNames[0] != "李四" {
		t.Fatalf("unexpected pending names: %+v", summary.RoundStatus.PendingPlayerNames)
	}
}
