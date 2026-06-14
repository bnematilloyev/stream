package viewers

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	activeKeyPrefix = "viewers:active:"
	uniqueKeyPrefix = "viewers:unique:"
	defaultWindow   = 45 * time.Second
)

// Stats holds concurrent and unique viewer counts.
type Stats struct {
	Concurrent int64
	Unique     int64
}

// Counter tracks live viewers using Redis ZSET (concurrent) + HyperLogLog (unique).
type Counter struct {
	redis  redis.Cmdable
	window time.Duration
}

func NewCounter(client redis.Cmdable, window time.Duration) *Counter {
	if window <= 0 {
		window = defaultWindow
	}
	return &Counter{redis: client, window: window}
}

// Heartbeat registers a viewer session for a live stream.
func (c *Counter) Heartbeat(ctx context.Context, streamID, sessionID string) (Stats, error) {
	if streamID == "" || sessionID == "" {
		return Stats{}, fmt.Errorf("stream_id and session_id required")
	}

	now := float64(time.Now().Unix())
	cutoff := fmt.Sprintf("%d", time.Now().Add(-c.window).Unix())
	activeKey := activeKeyPrefix + streamID
	uniqueKey := uniqueKeyPrefix + streamID

	pipe := c.redis.Pipeline()
	pipe.ZAdd(ctx, activeKey, redis.Z{Score: now, Member: sessionID})
	pipe.ZRemRangeByScore(ctx, activeKey, "-inf", cutoff)
	pipe.PFAdd(ctx, uniqueKey, sessionID)
	pipe.Expire(ctx, activeKey, c.window*3)
	pipe.Expire(ctx, uniqueKey, 24*time.Hour)
	if _, err := pipe.Exec(ctx); err != nil {
		return Stats{}, err
	}

	return c.Count(ctx, streamID)
}

// Count returns current concurrent and unique viewer stats.
func (c *Counter) Count(ctx context.Context, streamID string) (Stats, error) {
	activeKey := activeKeyPrefix + streamID
	uniqueKey := uniqueKeyPrefix + streamID
	cutoff := fmt.Sprintf("%d", time.Now().Add(-c.window).Unix())

	pipe := c.redis.Pipeline()
	zrem := pipe.ZRemRangeByScore(ctx, activeKey, "-inf", cutoff)
	concurrent := pipe.ZCard(ctx, activeKey)
	unique := pipe.PFCount(ctx, uniqueKey)
	if _, err := pipe.Exec(ctx); err != nil {
		return Stats{}, err
	}
	_ = zrem

	cc, err := concurrent.Result()
	if err != nil {
		return Stats{}, err
	}
	uc, err := unique.Result()
	if err != nil {
		return Stats{}, err
	}
	return Stats{Concurrent: cc, Unique: uc}, nil
}

// Clear removes viewer keys when a stream ends.
func (c *Counter) Clear(ctx context.Context, streamID string) error {
	return c.redis.Del(ctx, activeKeyPrefix+streamID, uniqueKeyPrefix+streamID).Err()
}
