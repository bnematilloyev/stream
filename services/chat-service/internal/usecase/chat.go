package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	apperrors "github.com/sahiy/sahiy-stream/pkg/errors"
	pkgnats "github.com/sahiy/sahiy-stream/pkg/nats"
	"github.com/sahiy/sahiy-stream/services/chat-service/internal/domain"
	"github.com/sahiy/sahiy-stream/services/chat-service/internal/moderation"
)

// OutboundEvent is broadcast to WebSocket clients and NATS.
type OutboundEvent struct {
	Type        string `json:"type"`
	ID          int64  `json:"id,omitempty"`
	StreamID    string `json:"stream_id,omitempty"`
	UserID      string `json:"user_id,omitempty"`
	Username    string `json:"username,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	Content     string `json:"content,omitempty"`
	MessageID   int64  `json:"message_id,omitempty"`
	TS          int64  `json:"ts,omitempty"`
}

type ChatUseCase struct {
	repo      domain.ChatRepository
	streams   domain.StreamChecker
	moderator domain.StreamModerator
	bus       *pkgnats.ChatBus
	limiter   *moderation.RateLimiter
}

func NewChatUseCase(
	repo domain.ChatRepository,
	streams domain.StreamChecker,
	moderator domain.StreamModerator,
	bus *pkgnats.ChatBus,
	limiter *moderation.RateLimiter,
) *ChatUseCase {
	return &ChatUseCase{repo: repo, streams: streams, moderator: moderator, bus: bus, limiter: limiter}
}

type SendInput struct {
	StreamID    uuid.UUID
	UserID      uuid.UUID
	Username    string
	DisplayName string
	Content     string
}

func (uc *ChatUseCase) Send(ctx context.Context, in SendInput) (OutboundEvent, error) {
	live, err := uc.streams.IsLive(ctx, in.StreamID)
	if err != nil {
		return OutboundEvent{}, apperrors.Internal(err)
	}
	if !live {
		return OutboundEvent{}, apperrors.NotFound("stream is not live")
	}

	banned, err := uc.repo.IsBanned(ctx, in.StreamID, in.UserID)
	if err != nil {
		return OutboundEvent{}, apperrors.Internal(err)
	}
	if banned {
		return OutboundEvent{}, apperrors.Forbidden("you are banned from this chat")
	}

	content, ok := moderation.Filter(in.Content)
	if !ok {
		return OutboundEvent{}, apperrors.Validation("invalid message content", nil)
	}

	allowed, err := uc.limiter.Allow(ctx, in.StreamID.String(), in.UserID.String())
	if err != nil {
		return OutboundEvent{}, apperrors.ServiceUnavailable("rate limit unavailable")
	}
	if !allowed {
		return OutboundEvent{}, apperrors.RateLimited()
	}

	msg := &domain.Message{
		StreamID:    in.StreamID,
		UserID:      &in.UserID,
		Username:    in.Username,
		DisplayName: in.DisplayName,
		Content:     content,
		Type:        domain.MsgTypeText,
		CreatedAt:   time.Now().UTC(),
	}
	if err := uc.repo.Insert(ctx, msg); err != nil {
		return OutboundEvent{}, apperrors.Internal(err)
	}

	event := OutboundEvent{
		Type:        "message",
		ID:          msg.ID,
		StreamID:    in.StreamID.String(),
		UserID:      in.UserID.String(),
		Username:    in.Username,
		DisplayName: in.DisplayName,
		Content:     content,
		TS:          msg.CreatedAt.Unix(),
	}
	if err := uc.broadcast(ctx, in.StreamID.String(), event); err != nil {
		return OutboundEvent{}, apperrors.Internal(err)
	}
	return event, nil
}

func (uc *ChatUseCase) GetHistory(ctx context.Context, streamID uuid.UUID, beforeID int64, limit int) ([]domain.Message, bool, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	list, err := uc.repo.ListHistory(ctx, streamID, beforeID, limit+1)
	if err != nil {
		return nil, false, apperrors.Internal(err)
	}
	hasMore := len(list) > limit
	if hasMore {
		list = list[:limit]
	}
	return list, hasMore, nil
}

func (uc *ChatUseCase) DeleteMessage(ctx context.Context, streamID uuid.UUID, messageID int64, actorUserID, actorRole string) error {
	if uc.moderator != nil {
		ok, err := uc.moderator.CanModerate(ctx, streamID, actorUserID, actorRole)
		if err != nil {
			return apperrors.Internal(err)
		}
		if !ok {
			return apperrors.Forbidden("access denied")
		}
	}

	if err := uc.repo.Delete(ctx, streamID, messageID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return apperrors.NotFound("message not found")
		}
		return apperrors.Internal(err)
	}
	return uc.broadcast(ctx, streamID.String(), OutboundEvent{
		Type:      "delete",
		MessageID: messageID,
		StreamID:  streamID.String(),
	})
}

func (uc *ChatUseCase) broadcast(ctx context.Context, streamID string, event OutboundEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return uc.bus.Publish(ctx, streamID, payload)
}
