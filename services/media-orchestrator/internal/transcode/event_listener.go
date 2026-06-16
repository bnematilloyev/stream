package transcode

import (
	"context"

	"github.com/google/uuid"
	pkgtranscode "github.com/sahiy/sahiy-stream/pkg/transcode"
	"github.com/sahiy/sahiy-stream/services/media-orchestrator/internal/domain"
	"go.uber.org/zap"
)

// EventListener updates stream media when workers report lifecycle events.
type EventListener struct {
	media domain.StreamMediaRepository
	log   *zap.Logger
}

func NewEventListener(media domain.StreamMediaRepository, log *zap.Logger) *EventListener {
	return &EventListener{media: media, log: log}
}

func (l *EventListener) Handle(evt pkgtranscode.JobEvent) error {
	switch evt.Type {
	case pkgtranscode.EventStarted:
		return l.updatePID(evt)
	case pkgtranscode.EventFailed:
		return l.markStopped(evt)
	default:
		return nil
	}
}

func (l *EventListener) updatePID(evt pkgtranscode.JobEvent) error {
	sid, err := uuid.Parse(evt.StreamID)
	if err != nil {
		return err
	}
	existing, err := l.media.GetByStreamID(context.Background(), sid)
	if err != nil || existing == nil {
		return err
	}
	pid := evt.FFmpegPID
	existing.Status = domain.StatusIngesting
	existing.FFmpegPID = &pid
	existing.StoppedAt = nil
	if err := l.media.Upsert(context.Background(), existing); err != nil {
		return err
	}
	l.log.Info("ffmpeg pid updated from worker",
		zap.String("stream_id", evt.StreamID),
		zap.Int("pid", pid),
		zap.String("worker_id", evt.WorkerID),
	)
	return nil
}

func (l *EventListener) markStopped(evt pkgtranscode.JobEvent) error {
	sid, err := uuid.Parse(evt.StreamID)
	if err != nil {
		return err
	}
	existing, err := l.media.GetByStreamID(context.Background(), sid)
	if err != nil || existing == nil {
		return err
	}
	existing.Status = domain.StatusStopped
	existing.StoppedAt = &evt.At
	if err := l.media.Upsert(context.Background(), existing); err != nil {
		return err
	}
	l.log.Warn("transcode did not start",
		zap.String("stream_id", evt.StreamID),
		zap.String("event", evt.Type),
		zap.String("worker_id", evt.WorkerID),
		zap.String("error", evt.Error),
	)
	return nil
}
