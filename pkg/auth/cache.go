package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	keyUserStatus  = "auth:user:status:%s"
	keyUserRevoke  = "auth:user:revoked:%s"
	keyUserProfile = "auth:user:profile:%s"
)

// SessionCache stores user auth state in Redis for fast gateway validation.
type SessionCache struct {
	client *redis.Client
	ttl    time.Duration
}

func NewSessionCache(client *redis.Client, ttl time.Duration) *SessionCache {
	return &SessionCache{client: client, ttl: ttl}
}

func (c *SessionCache) SetUserActive(ctx context.Context, user *Principal) error {
	if err := c.client.Set(ctx, fmt.Sprintf(keyUserStatus, user.ID), user.Status, c.ttl).Err(); err != nil {
		return err
	}
	return c.cacheProfile(ctx, user)
}

func (c *SessionCache) GetUserStatus(ctx context.Context, userID string) (string, error) {
	val, err := c.client.Get(ctx, fmt.Sprintf(keyUserStatus, userID)).Result()
	if err == redis.Nil {
		return "", nil
	}
	return val, err
}

func (c *SessionCache) GetUserProfile(ctx context.Context, userID string) (*Principal, error) {
	raw, err := c.client.Get(ctx, fmt.Sprintf(keyUserProfile, userID)).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var user Principal
	if err := json.Unmarshal([]byte(raw), &user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (c *SessionCache) CacheUserStatus(ctx context.Context, userID, status string) error {
	return c.client.Set(ctx, fmt.Sprintf(keyUserStatus, userID), status, c.ttl).Err()
}

func (c *SessionCache) cacheProfile(ctx context.Context, user *Principal) error {
	raw, err := json.Marshal(user)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, fmt.Sprintf(keyUserProfile, user.ID), raw, c.ttl).Err()
}

func (c *SessionCache) RevokeUser(ctx context.Context, userID string, ttl time.Duration) error {
	pipe := c.client.Pipeline()
	pipe.Set(ctx, fmt.Sprintf(keyUserRevoke, userID), "1", ttl)
	pipe.Del(ctx, fmt.Sprintf(keyUserProfile, userID))
	pipe.Del(ctx, fmt.Sprintf(keyUserStatus, userID))
	_, err := pipe.Exec(ctx)
	return err
}

func (c *SessionCache) ClearUserRevoke(ctx context.Context, userID string) error {
	return c.client.Del(ctx, fmt.Sprintf(keyUserRevoke, userID)).Err()
}

func (c *SessionCache) IsUserRevoked(ctx context.Context, userID string) (bool, error) {
	n, err := c.client.Exists(ctx, fmt.Sprintf(keyUserRevoke, userID)).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func PrincipalFromDomain(id, email, username, displayName, role, status string, emailVerified bool, createdAt time.Time) *Principal {
	return &Principal{
		ID:            id,
		Email:         email,
		Username:      username,
		DisplayName:   displayName,
		Role:          role,
		Status:        status,
		EmailVerified: emailVerified,
		CreatedAt:     createdAt,
	}
}
