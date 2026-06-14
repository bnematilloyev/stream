package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sahiy/sahiy-stream/pkg/auth"
	"github.com/sahiy/sahiy-stream/services/auth-service/internal/domain"
)

type sessionCache interface {
	SetUserActive(ctx context.Context, user *auth.Principal) error
	RevokeUser(ctx context.Context, userID string, ttl time.Duration) error
	ClearUserRevoke(ctx context.Context, userID string) error
}

func (uc *AuthUseCase) cacheUserSession(ctx context.Context, user *domain.User) {
	if uc.cache == nil {
		return
	}
	_ = uc.cache.SetUserActive(ctx, auth.PrincipalFromDomain(
		user.ID.String(), user.Email, user.Username, user.DisplayName,
		user.Role, user.Status, user.EmailVerified, user.CreatedAt,
	))
	_ = uc.cache.ClearUserRevoke(ctx, user.ID.String())
}

func (uc *AuthUseCase) revokeUserSession(ctx context.Context, userID uuid.UUID) {
	if uc.cache == nil {
		return
	}
	_ = uc.cache.RevokeUser(ctx, userID.String(), uc.jwt.AccessTTL())
}
