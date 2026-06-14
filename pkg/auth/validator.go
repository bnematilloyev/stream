package auth

import (
	"context"

	apperrors "github.com/sahiy/sahiy-stream/pkg/errors"
	"github.com/sahiy/sahiy-stream/pkg/crypto"
)

// UserFetcher loads full user details on cache miss (Adapter pattern).
type UserFetcher interface {
	FetchUser(ctx context.Context, userID string) (*Principal, error)
}

// Validator performs local JWT validation with Redis-backed session checks.
type Validator struct {
	jwt     *crypto.JWTManager
	cache   *SessionCache
	fetcher UserFetcher
}

func NewValidator(jwt *crypto.JWTManager, cache *SessionCache, fetcher UserFetcher) *Validator {
	return &Validator{jwt: jwt, cache: cache, fetcher: fetcher}
}

func (v *Validator) ValidateAccess(ctx context.Context, accessToken string) (*Principal, error) {
	claims, err := v.jwt.ValidateAccess(accessToken)
	if err != nil {
		return nil, apperrors.New(apperrors.CodeTokenInvalid, "invalid or expired token", 401)
	}

	revoked, err := v.cache.IsUserRevoked(ctx, claims.UserID)
	if err != nil {
		return nil, apperrors.ServiceUnavailable("auth cache unavailable")
	}
	if revoked {
		return nil, apperrors.Unauthorized("invalid or expired token")
	}

	if profile, err := v.cache.GetUserProfile(ctx, claims.UserID); err != nil {
		return nil, apperrors.ServiceUnavailable("auth cache unavailable")
	} else if profile != nil && profile.IsActive() {
		return profile, nil
	}

	status, err := v.cache.GetUserStatus(ctx, claims.UserID)
	if err != nil {
		return nil, apperrors.ServiceUnavailable("auth cache unavailable")
	}
	if status != "" && status != StatusActive {
		return nil, apperrors.Forbidden("account is not active")
	}

	return v.fetchAndCache(ctx, claims.UserID)
}

func (v *Validator) fetchAndCache(ctx context.Context, userID string) (*Principal, error) {
	if v.fetcher == nil {
		return nil, apperrors.Internal(nil)
	}

	user, err := v.fetcher.FetchUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil || !user.IsActive() {
		return nil, apperrors.Forbidden("account is not active")
	}

	if err := v.cache.SetUserActive(ctx, user); err != nil {
		return nil, apperrors.ServiceUnavailable("auth cache unavailable")
	}
	return user, nil
}
