package usecase

import (
	"context"

	"github.com/google/uuid"
	apperrors "github.com/sahiy/sahiy-stream/pkg/errors"
	"github.com/sahiy/sahiy-stream/pkg/analytics"
	"github.com/sahiy/sahiy-stream/pkg/viewers"
	"github.com/sahiy/sahiy-stream/services/stream-service/internal/domain"
)

type ViewerUseCase struct {
	streams   domain.StreamRepository
	counter   *viewers.Counter
	analytics *analytics.Client
}

func NewViewerUseCase(streams domain.StreamRepository, counter *viewers.Counter, analytics *analytics.Client) *ViewerUseCase {
	return &ViewerUseCase{streams: streams, counter: counter, analytics: analytics}
}

func (uc *ViewerUseCase) Heartbeat(ctx context.Context, streamID uuid.UUID, sessionID string) (viewers.Stats, error) {
	st, err := uc.streams.GetByID(ctx, streamID)
	if err != nil {
		return viewers.Stats{}, apperrors.Internal(err)
	}
	if st == nil || st.Status != domain.StatusLive {
		return viewers.Stats{}, apperrors.NotFound("stream is not live")
	}

	stats, err := uc.counter.Heartbeat(ctx, streamID.String(), sessionID)
	if err != nil {
		return viewers.Stats{}, apperrors.Internal(err)
	}

	if uc.analytics != nil && uc.analytics.Enabled() {
		_ = uc.analytics.RecordViewerHeartbeat(ctx, analytics.ViewerHeartbeatEvent{
			StreamID:   streamID.String(),
			SessionID:  sessionID,
			Concurrent: stats.Concurrent,
			Unique:     stats.Unique,
		})
	}
	return stats, nil
}

func (uc *ViewerUseCase) Stats(ctx context.Context, streamID uuid.UUID) (viewers.Stats, error) {
	stats, err := uc.counter.Count(ctx, streamID.String())
	if err != nil {
		return viewers.Stats{}, apperrors.Internal(err)
	}
	return stats, nil
}

func (uc *ViewerUseCase) EnrichStream(ctx context.Context, st *domain.Stream) {
	if st == nil || uc.counter == nil {
		return
	}
	stats, err := uc.counter.Count(ctx, st.ID.String())
	if err != nil {
		return
	}
	st.ViewerCount = int(stats.Concurrent)
	if int(stats.Unique) > st.PeakViewers {
		st.PeakViewers = int(stats.Unique)
	}
}

func (uc *ViewerUseCase) EnrichStreams(ctx context.Context, list []domain.Stream) {
	for i := range list {
		uc.EnrichStream(ctx, &list[i])
	}
}

func (uc *ViewerUseCase) ClearStream(ctx context.Context, streamID uuid.UUID) {
	if uc.counter != nil {
		_ = uc.counter.Clear(ctx, streamID.String())
	}
}
