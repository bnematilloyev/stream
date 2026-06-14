package moderation

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const defaultRateLimit = 5

// RateLimiter limits messages per user per stream (per second).
type RateLimiter struct {
	redis redis.Cmdable
	limit int
}

func NewRateLimiter(client redis.Cmdable, limit int) *RateLimiter {
	if limit <= 0 {
		limit = defaultRateLimit
	}
	return &RateLimiter{redis: client, limit: limit}
}

func (r *RateLimiter) Allow(ctx context.Context, streamID, userID string) (bool, error) {
	key := fmt.Sprintf("chat:rate:%s:%s", streamID, userID)
	count, err := r.redis.Incr(ctx, key).Result()
	if err != nil {
		return false, err
	}
	if count == 1 {
		_ = r.redis.Expire(ctx, key, time.Second).Err()
	}
	return count <= int64(r.limit), nil
}
