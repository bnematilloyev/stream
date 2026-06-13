package usecase

import (
	"context"
	"strings"

	"github.com/google/uuid"
	apperrors "github.com/sahiy/sahiy-stream/pkg/errors"
	"github.com/sahiy/sahiy-stream/services/user-service/internal/domain"
)

type UserUseCase struct {
	profiles domain.ProfileRepository
}

func NewUserUseCase(profiles domain.ProfileRepository) *UserUseCase {
	return &UserUseCase{profiles: profiles}
}

func (uc *UserUseCase) GetProfile(ctx context.Context, userID uuid.UUID) (*domain.Profile, error) {
	p, err := uc.profiles.GetByID(ctx, userID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if p == nil {
		return nil, apperrors.NotFound("user not found")
	}
	return p, nil
}

func (uc *UserUseCase) GetPublicProfile(ctx context.Context, username string) (*domain.Profile, error) {
	username = strings.TrimSpace(username)
	p, err := uc.profiles.GetByUsername(ctx, username)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if p == nil {
		return nil, apperrors.NotFound("user not found")
	}
	return p, nil
}

func (uc *UserUseCase) UpdateProfile(ctx context.Context, userID uuid.UUID, displayName, avatarURL *string) (*domain.Profile, error) {
	if displayName != nil {
		name := strings.TrimSpace(*displayName)
		if len(name) < 2 || len(name) > 100 {
			return nil, apperrors.Validation("display_name must be 2-100 characters", nil)
		}
		displayName = &name
	}
	p, err := uc.profiles.Update(ctx, userID, displayName, avatarURL)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if p == nil {
		return nil, apperrors.NotFound("user not found")
	}
	return p, nil
}
