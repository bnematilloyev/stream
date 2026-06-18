package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
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
	UpdatedAt   time.Time
}

const (
	MediaStatusIdle      = "idle"
	MediaStatusIngesting = "ingesting"
	MediaStatusReady     = "ready"
	MediaStatusStopped   = "stopped"
)

type StreamMediaRepository interface {
	GetByStreamID(ctx context.Context, streamID uuid.UUID) (*StreamMedia, error)
	Upsert(ctx context.Context, m *StreamMedia) error
}
