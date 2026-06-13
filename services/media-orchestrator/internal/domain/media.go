package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

const (
	StatusIdle      = "idle"
	StatusIngesting = "ingesting"
	StatusReady     = "ready"
	StatusStopped   = "stopped"
)

type StreamMedia struct {
	StreamID    uuid.UUID
	Status      string
	HLSPath     *string
	PlaybackURL *string
	IngestName  *string
	FFmpegPID   *int
	StartedAt   *time.Time
	StoppedAt   *time.Time
}

type StreamMediaRepository interface {
	Upsert(ctx context.Context, m *StreamMedia) error
	GetByStreamID(ctx context.Context, streamID uuid.UUID) (*StreamMedia, error)
}
