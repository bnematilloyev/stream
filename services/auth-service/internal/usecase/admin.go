package usecase

import (
	"context"

	"github.com/google/uuid"
	apperrors "github.com/sahiy/sahiy-stream/pkg/errors"
	"github.com/sahiy/sahiy-stream/services/auth-service/internal/domain"
)

func (uc *AuthUseCase) ListUsers(ctx context.Context, status, role, search string, page, limit int) ([]domain.User, int, error) {
	return uc.users.List(ctx, status, role, search, page, limit)
}

func (uc *AuthUseCase) UpdateUserAdmin(ctx context.Context, actorID, userID uuid.UUID, role, status *string) (*domain.User, error) {
	if role != nil {
		switch *role {
		case domain.RoleUser, domain.RoleModerator, domain.RoleAdmin:
		default:
			return nil, apperrors.Validation("invalid role", map[string]any{"allowed": []string{domain.RoleUser, domain.RoleModerator, domain.RoleAdmin}})
		}
	}
	if status != nil {
		switch *status {
		case domain.StatusActive, domain.StatusSuspended, domain.StatusBanned:
		default:
			return nil, apperrors.Validation("invalid status", map[string]any{"allowed": []string{domain.StatusActive, domain.StatusSuspended, domain.StatusBanned}})
		}
	}

	user, err := uc.users.UpdateAdmin(ctx, userID, role, status)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if user == nil {
		return nil, apperrors.NotFound("user not found")
	}

	actor := &actorID
	_ = uc.audit.Log(ctx, actor, "admin.user.update", "user", &userID, map[string]any{
		"role": role, "status": status,
	}, nil)
	if status != nil && *status != domain.StatusActive {
		_ = uc.sessions.DeleteByUserID(ctx, userID)
	}
	return user, nil
}

func (uc *AuthUseCase) GetPlatformStats(ctx context.Context) (int64, map[string]int64, int64, error) {
	return uc.users.CountByStatus(ctx)
}

func (uc *AuthUseCase) ListAuditLogs(ctx context.Context, page, limit int) ([]domain.AuditLog, int, error) {
	return uc.audit.List(ctx, page, limit)
}
