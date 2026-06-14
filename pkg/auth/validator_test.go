package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/sahiy/sahiy-stream/pkg/auth"
	"github.com/sahiy/sahiy-stream/pkg/crypto"
)

type stubFetcher struct {
	user *auth.Principal
}

func (s stubFetcher) FetchUser(_ context.Context, _ string) (*auth.Principal, error) {
	return s.user, nil
}

func TestValidatorValidateAccess(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	jwtManager := crypto.NewJWTManager(
		"test-access-secret-minimum-32-characters",
		"test-refresh-secret-minimum-32-characters",
		15*time.Minute,
		168*time.Hour,
	)
	cache := auth.NewSessionCache(redisClient, 5*time.Minute)

	user := &auth.Principal{
		ID: "11111111-1111-1111-1111-111111111111", Email: "user@test.com",
		Username: "user1", DisplayName: "User One", Role: "user", Status: auth.StatusActive,
		CreatedAt: time.Now().UTC(),
	}
	if err := cache.SetUserActive(context.Background(), user); err != nil {
		t.Fatal(err)
	}

	pair, err := jwtManager.GeneratePair(user.ID, user.Username, user.Role)
	if err != nil {
		t.Fatal(err)
	}

	validator := auth.NewValidator(jwtManager, cache, stubFetcher{user: user})
	got, err := validator.ValidateAccess(context.Background(), pair.AccessToken)
	if err != nil {
		t.Fatalf("ValidateAccess() error = %v", err)
	}
	if got.ID != user.ID || got.Email != user.Email {
		t.Fatalf("unexpected principal: %+v", got)
	}
}

func TestValidatorRejectsRevokedUser(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	jwtManager := crypto.NewJWTManager(
		"test-access-secret-minimum-32-characters",
		"test-refresh-secret-minimum-32-characters",
		15*time.Minute,
		168*time.Hour,
	)
	cache := auth.NewSessionCache(redisClient, 5*time.Minute)

	userID := "11111111-1111-1111-1111-111111111111"
	pair, err := jwtManager.GeneratePair(userID, "user1", "user")
	if err != nil {
		t.Fatal(err)
	}
	if err := cache.RevokeUser(context.Background(), userID, time.Minute); err != nil {
		t.Fatal(err)
	}

	validator := auth.NewValidator(jwtManager, cache, stubFetcher{})
	if _, err := validator.ValidateAccess(context.Background(), pair.AccessToken); err == nil {
		t.Fatal("expected revoked user to be rejected")
	}
}
