package domain

import (
	"context"

	"github.com/google/uuid"
)

// StreamModerator checks whether a user may moderate a stream's chat.
type StreamModerator interface {
	CanModerate(ctx context.Context, streamID uuid.UUID, userID, role string) (bool, error)
}
