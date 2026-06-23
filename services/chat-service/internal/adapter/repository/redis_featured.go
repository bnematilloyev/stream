package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/sahiy/sahiy-stream/services/chat-service/internal/domain"
)

// featuredTTL bounds how long a spotlight survives without refresh. It outlives
// a typical broadcast but self-expires so abandoned streams don't leak state.
const featuredTTL = 12 * time.Hour

type RedisFeaturedStore struct {
	client *redis.Client
}

func NewRedisFeaturedStore(client *redis.Client) *RedisFeaturedStore {
	return &RedisFeaturedStore{client: client}
}

func featuredKey(streamID uuid.UUID) string {
	return "chat:featured:" + streamID.String()
}

func (s *RedisFeaturedStore) Set(ctx context.Context, streamID uuid.UUID, product domain.FeaturedProduct) error {
	payload, err := json.Marshal(product)
	if err != nil {
		return fmt.Errorf("marshal featured product: %w", err)
	}
	return s.client.Set(ctx, featuredKey(streamID), payload, featuredTTL).Err()
}

func (s *RedisFeaturedStore) Get(ctx context.Context, streamID uuid.UUID) (*domain.FeaturedProduct, error) {
	raw, err := s.client.Get(ctx, featuredKey(streamID)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var product domain.FeaturedProduct
	if err := json.Unmarshal(raw, &product); err != nil {
		return nil, fmt.Errorf("unmarshal featured product: %w", err)
	}
	return &product, nil
}

func (s *RedisFeaturedStore) Clear(ctx context.Context, streamID uuid.UUID) error {
	return s.client.Del(ctx, featuredKey(streamID)).Err()
}

var _ domain.FeaturedStore = (*RedisFeaturedStore)(nil)
