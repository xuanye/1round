package scheduler

import (
	"context"
	"log/slog"
	"time"

	settlementsvc "github.com/xuanye/one-round/apps/server/internal/app/settlement"
)

// AutoSettlementRunner periodically scans for inactive games and settles them.
type AutoSettlementRunner struct {
	service   *settlementsvc.Service
	interval  time.Duration
	threshold time.Duration
	logger    *slog.Logger
}

func NewAutoSettlementRunner(service *settlementsvc.Service, logger *slog.Logger, interval, threshold time.Duration) *AutoSettlementRunner {
	return &AutoSettlementRunner{
		service:   service,
		interval:  interval,
		threshold: threshold,
		logger:    logger,
	}
}

// Start begins the periodic auto-settlement scan in a background goroutine.
// It stops when the provided context is cancelled.
func (r *AutoSettlementRunner) Start(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				result, err := r.service.SettleInactiveGames(ctx, r.threshold)
				if err != nil {
					r.logger.Error("auto settlement failed", "error", err)
					continue
				}
				if result.Finished > 0 || result.Voided > 0 {
					r.logger.Info("auto settlement completed", "finished", result.Finished, "voided", result.Voided)
				}
			}
		}
	}()
}
