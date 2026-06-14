package worker

import (
	"context"
	"time"

	"github.com/sahiy/sahiy-stream/pkg/viewers"
	"github.com/sahiy/sahiy-stream/services/stream-service/internal/domain"
	"go.uber.org/zap"
)

// ViewerSyncWorker persists Redis viewer stats to PostgreSQL.
type ViewerSyncWorker struct {
	streams  domain.StreamRepository
	counter  *viewers.Counter
	interval time.Duration
	log      *zap.Logger
}

func NewViewerSyncWorker(streams domain.StreamRepository, counter *viewers.Counter, interval time.Duration, log *zap.Logger) *ViewerSyncWorker {
	if interval <= 0 {
		interval = 15 * time.Second
	}
	return &ViewerSyncWorker{streams: streams, counter: counter, interval: interval, log: log}
}

func (w *ViewerSyncWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.syncOnce(ctx)
		}
	}
}

func (w *ViewerSyncWorker) syncOnce(ctx context.Context) {
	ids, err := w.streams.ListLiveStreamIDs(ctx)
	if err != nil {
		w.log.Warn("list live streams failed", zap.Error(err))
		return
	}
	for _, id := range ids {
		stats, err := w.counter.Count(ctx, id.String())
		if err != nil {
			w.log.Warn("viewer count failed", zap.String("stream_id", id.String()), zap.Error(err))
			continue
		}
		if err := w.streams.UpdateViewerStats(ctx, id, int(stats.Concurrent), int(stats.Unique)); err != nil {
			w.log.Warn("viewer sync failed", zap.String("stream_id", id.String()), zap.Error(err))
		}
	}
}
