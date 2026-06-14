package transcode

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	pkgnats "github.com/sahiy/sahiy-stream/pkg/nats"
	pkgtranscode "github.com/sahiy/sahiy-stream/pkg/transcode"
)

// QueueBackend dispatches transcode jobs via NATS JetStream.
type QueueBackend struct {
	bus      *pkgnats.TranscodeBus
	workerID string
}

func NewQueueBackend(bus *pkgnats.TranscodeBus) *QueueBackend {
	return &QueueBackend{bus: bus}
}

func (b *QueueBackend) Start(ctx context.Context, req StartRequest) (*RunningJob, error) {
	jobID := uuid.NewString()
	job := pkgtranscode.StartJob{
		JobID: jobID, StreamID: req.StreamID, IngestName: req.IngestName,
		InputURL: req.InputURL, OutputDir: req.OutputDir, LatencyMode: req.LatencyMode,
		Quality: pkgtranscode.NormalizeQuality(req.Quality), Encoder: req.Encoder,
		Storage: req.Storage, Priority: 10,
		IssuedAt: time.Now().UTC(),
	}
	if err := b.bus.PublishStart(ctx, job); err != nil {
		return nil, fmt.Errorf("publish start job: %w", err)
	}
	return &RunningJob{JobID: jobID, StreamID: req.StreamID}, nil
}

func (b *QueueBackend) Stop(ctx context.Context, streamID, reason string) error {
	return b.bus.PublishStop(ctx, pkgtranscode.StopJob{
		JobID: streamID, StreamID: streamID, Reason: reason,
	})
}

// Bus exposes the underlying NATS bus for event subscription.
func (b *QueueBackend) Bus() *pkgnats.TranscodeBus { return b.bus }
