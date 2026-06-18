package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sahiy/sahiy-stream/pkg/pagination"
)

const (
	StatusScheduled  = "scheduled"
	StatusLive       = "live"
	StatusEnded      = "ended"
	StatusProcessing = "processing"
	StatusReady      = "ready"
)

type Stream struct {
	ID                  uuid.UUID
	ChannelID           uuid.UUID
	ChannelSlug         string
	ChannelTitle        string
	Title               string
	Description         *string
	ThumbnailURL        *string
	Status              string
	IngestProtocol      string
	LatencyMode         string
	Visibility          string
	CategoryID          *uuid.UUID
	Tags                []string
	ScheduledAt         *time.Time
	StartedAt           *time.Time
	EndedAt             *time.Time
	ViewerCount         int
	PeakViewers         int
	MarketplaceSellerID *int64
	MarketplaceShopID   *int64
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type Channel struct {
	ID     uuid.UUID
	UserID uuid.UUID
	Slug   string
	Title  string
}

type StreamKey struct {
	ID        uuid.UUID
	ChannelID uuid.UUID
	KeyLookup string
}

type StreamRepository interface {
	Create(ctx context.Context, s *Stream) error
	GetByID(ctx context.Context, id uuid.UUID) (*Stream, error)
	Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, title, description, visibility *string, categoryID *uuid.UUID, tags []string) (*Stream, error)
	Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	ListLive(ctx context.Context, p pagination.Params) ([]Stream, int, error)
	ListMarketplaceLive(ctx context.Context, p pagination.Params) ([]Stream, int, error)
	EndStaleLive(ctx context.Context) error
	ReconcileLiveStream(ctx context.Context, id uuid.UUID) error
	ListByChannel(ctx context.Context, channelID uuid.UUID, status string, p pagination.Params) ([]Stream, int, error)
	SetStatus(ctx context.Context, id uuid.UUID, status string, startedAt, endedAt *time.Time) error
	GetActiveLiveByChannel(ctx context.Context, channelID uuid.UUID) (*Stream, error)
	GetActiveLiveByChannelAndProtocol(ctx context.Context, channelID uuid.UUID, ingestProtocol string) (*Stream, error)
	CountLiveByChannel(ctx context.Context, channelID uuid.UUID) (int, error)
	GetLatestScheduledByChannel(ctx context.Context, channelID uuid.UUID) (*Stream, error)
	GetLatestScheduledByChannelAndProtocol(ctx context.Context, channelID uuid.UUID, ingestProtocol string) (*Stream, error)
	UpdateViewerStats(ctx context.Context, id uuid.UUID, concurrent, unique int) error
	ListLiveStreamIDs(ctx context.Context) ([]uuid.UUID, error)
}

type ChannelRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*Channel, error)
	GetBySlug(ctx context.Context, slug string) (*Channel, error)
	SetLive(ctx context.Context, channelID uuid.UUID, live bool) error
}

type StreamKeyRepository interface {
	GetByLookup(ctx context.Context, lookup string) (*StreamKey, error)
	UpdateLastUsed(ctx context.Context, id uuid.UUID) error
}
