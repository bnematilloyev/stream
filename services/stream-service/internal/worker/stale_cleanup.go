package worker

import (
	"context"
	"time"

	"github.com/sahiy/sahiy-stream/services/stream-service/internal/domain"
	"go.uber.org/zap"
)

// StaleCleanupWorker periodically ends streams marked live without active ingest.
type StaleCleanupWorker struct {
	repo     domain.StreamRepository
	interval time.Duration
	log      *zap.Logger
}

func NewStaleCleanupWorker(repo domain.StreamRepository, interval time.Duration, log *zap.Logger) *StaleCleanupWorker {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	return &StaleCleanupWorker{repo: repo, interval: interval, log: log}
}

func (w *StaleCleanupWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.repo.EndStaleLive(ctx); err != nil {
				w.log.Warn("stale live cleanup failed", zap.Error(err))
			}
		}
	}
}
