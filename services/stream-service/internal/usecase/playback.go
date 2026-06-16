package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	apperrors "github.com/sahiy/sahiy-stream/pkg/errors"
	"github.com/sahiy/sahiy-stream/pkg/playback"
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
	streams      domain.StreamRepository
	media        domain.StreamMediaRepository
	signer       *playback.Signer
	playbackBase string
}

func NewPlaybackUseCase(
	streams domain.StreamRepository,
	media domain.StreamMediaRepository,
	signer *playback.Signer,
	playbackBase string,
) *PlaybackUseCase {
	return &PlaybackUseCase{
		streams:      streams,
		media:        media,
		signer:       signer,
		playbackBase: strings.TrimRight(playbackBase, "/"),
	}
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

	status := domain.MediaStatusIdle
	if m != nil {
		status = m.Status
	}

	if st.Status != domain.StatusLive {
		return nil, apperrors.NotFound("playback not available")
	}
	if m == nil || (m.Status != domain.MediaStatusIngesting && m.Status != domain.MediaStatusReady) {
		return nil, apperrors.NotFound("playback not available")
	}

	resource := "master.m3u8"
	unsigned := fmt.Sprintf("%s/playback/%s", uc.playbackBase, streamID.String())
	signedURL, expiresAt := uc.signer.Sign(unsigned, streamID.String(), resource)

	return &PlaybackResult{
		StreamID:  streamID,
		URL:       signedURL,
		Format:    "hls",
		Status:    status,
		ExpiresAt: expiresAt,
	}, nil
}

func (uc *PlaybackUseCase) UpsertMedia(ctx context.Context, m *domain.StreamMedia) error {
	return uc.media.Upsert(ctx, m)
}
