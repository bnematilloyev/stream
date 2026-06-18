package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	apperrors "github.com/sahiy/sahiy-stream/pkg/errors"
	"github.com/sahiy/sahiy-stream/services/stream-service/internal/domain"
)

func (uc *StreamUseCase) GetScheduledForChannel(ctx context.Context, channelID uuid.UUID) (*domain.Stream, error) {
	return uc.getScheduledForChannelProtocol(ctx, channelID, "rtmp")
}

func (uc *StreamUseCase) getScheduledForChannelProtocol(ctx context.Context, channelID uuid.UUID, ingestProtocol string) (*domain.Stream, error) {
	live, err := uc.streams.GetActiveLiveByChannelAndProtocol(ctx, channelID, ingestProtocol)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if live != nil {
		return live, nil
	}
	scheduled, err := uc.streams.GetLatestScheduledByChannelAndProtocol(ctx, channelID, ingestProtocol)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if scheduled == nil {
		return nil, apperrors.NotFound("no scheduled stream for channel")
	}
	return scheduled, nil
}

func (uc *StreamUseCase) StartIngest(ctx context.Context, channelID uuid.UUID) (*domain.Stream, error) {
	st, err := uc.GetScheduledForChannel(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if st.Status == domain.StatusLive {
		return st, nil
	}
	now := time.Now()
	if err := uc.streams.SetStatus(ctx, st.ID, domain.StatusLive, &now, nil); err != nil {
		return nil, apperrors.Internal(err)
	}
	_ = uc.channels.SetLive(ctx, channelID, true)
	return uc.streams.GetByID(ctx, st.ID)
}

func (uc *StreamUseCase) StartIngestStream(ctx context.Context, streamID uuid.UUID) (*domain.Stream, error) {
	st, err := uc.streams.GetByID(ctx, streamID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if st == nil {
		return nil, apperrors.NotFound("stream not found")
	}
	if st.Status == domain.StatusLive {
		return st, nil
	}
	if st.Status != domain.StatusScheduled {
		return nil, apperrors.Validation("stream not publishable", nil)
	}
	now := time.Now()
	if err := uc.streams.SetStatus(ctx, streamID, domain.StatusLive, &now, nil); err != nil {
		return nil, apperrors.Internal(err)
	}
	_ = uc.channels.SetLive(ctx, st.ChannelID, true)
	return uc.streams.GetByID(ctx, streamID)
}

func (uc *StreamUseCase) EndIngest(ctx context.Context, streamID uuid.UUID) (*domain.Stream, error) {
	st, err := uc.streams.GetByID(ctx, streamID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if st == nil {
		return nil, apperrors.NotFound("stream not found")
	}
	if st.Status != domain.StatusLive {
		return st, nil
	}
	now := time.Now()
	if err := uc.streams.SetStatus(ctx, streamID, domain.StatusEnded, nil, &now); err != nil {
		return nil, apperrors.Internal(err)
	}
	count, err := uc.streams.CountLiveByChannel(ctx, st.ChannelID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if count == 0 {
		_ = uc.channels.SetLive(ctx, st.ChannelID, false)
	}
	if uc.viewers != nil {
		uc.viewers.ClearStream(ctx, streamID)
	}
	return uc.streams.GetByID(ctx, streamID)
}
