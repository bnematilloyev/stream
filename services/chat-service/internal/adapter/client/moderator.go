package client

import (
	"context"

	"github.com/google/uuid"
	"github.com/sahiy/sahiy-stream/services/chat-service/internal/domain"
)

type StreamModerator struct {
	streams *StreamClient
	users   *UserClient
}

func NewStreamModerator(streams *StreamClient, users *UserClient) *StreamModerator {
	return &StreamModerator{streams: streams, users: users}
}

func (m *StreamModerator) CanModerate(ctx context.Context, streamID uuid.UUID, userID, role string) (bool, error) {
	if role == "admin" {
		return true, nil
	}
	st, err := m.streams.GetStream(ctx, streamID)
	if err != nil {
		return false, err
	}
	ch, err := m.users.GetChannel(ctx, st.GetChannelSlug())
	if err != nil {
		return false, err
	}
	return ch.GetUserId() == userID, nil
}

var _ domain.StreamModerator = (*StreamModerator)(nil)
