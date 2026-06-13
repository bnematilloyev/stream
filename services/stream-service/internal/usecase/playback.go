package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	apperrors "github.com/sahiy/sahiy-stream/pkg/errors"
	"github.com/sahiy/sahiy-stream/services/stream-service/internal/domain"
)

type PlaybackResult struct {
	StreamID  uuid.UUID
	URL       string
	Format    string
	Status    string
	ExpiresAt time.Time
}

type PlaybackUseCase struct {
	streams domain.StreamRepository
	media   domain.StreamMediaRepository
	hlsBase string
}

func NewPlaybackUseCase(streams domain.StreamRepository, media domain.StreamMediaRepository, hlsBase string) *PlaybackUseCase {
	return &PlaybackUseCase{streams: streams, media: media, hlsBase: hlsBase}
}

func (uc *PlaybackUseCase) GetPlayback(ctx context.Context, streamID uuid.UUID) (*PlaybackResult, error) {
	st, err := uc.streams.GetByID(ctx, streamID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if st == nil {
		return nil, apperrors.NotFound("stream not found")
	}

	m, err := uc.media.GetByStreamID(ctx, streamID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	url := fmt.Sprintf("%s/%s/master.m3u8", uc.hlsBase, streamID.String())
	status := domain.MediaStatusIdle
	if m != nil {
		status = m.Status
		if m.PlaybackURL != nil && *m.PlaybackURL != "" {
			url = *m.PlaybackURL
		}
	}

	if st.Status != domain.StatusLive {
		return nil, apperrors.NotFound("playback not available")
	}
	if m == nil || m.Status != domain.MediaStatusIngesting {
		return nil, apperrors.NotFound("playback not available")
	}

	return &PlaybackResult{
		StreamID:  streamID,
		URL:       url,
		Format:    "hls",
		Status:    status,
		ExpiresAt: time.Now().Add(4 * time.Hour),
	}, nil
}

func (uc *PlaybackUseCase) UpsertMedia(ctx context.Context, m *domain.StreamMedia) error {
	return uc.media.Upsert(ctx, m)
}
