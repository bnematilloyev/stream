package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

const (
	MsgTypeText   = "text"
	MsgTypeSystem = "system"
)

type Message struct {
	ID          int64
	StreamID    uuid.UUID
	UserID      *uuid.UUID
	Username    string
	DisplayName string
	Content     string
	Type        string
	CreatedAt   time.Time
}

type ChatRepository interface {
	Insert(ctx context.Context, msg *Message) error
	ListHistory(ctx context.Context, streamID uuid.UUID, beforeID int64, limit int) ([]Message, error)
	Delete(ctx context.Context, streamID uuid.UUID, messageID int64) error
	IsBanned(ctx context.Context, streamID, userID uuid.UUID) (bool, error)
}

type StreamChecker interface {
	IsLive(ctx context.Context, streamID uuid.UUID) (bool, error)
}
