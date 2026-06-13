package grpc

import (
	"context"

	"github.com/google/uuid"
	userv1 "github.com/sahiy/sahiy-stream/proto/gen/user/v1"
	"github.com/sahiy/sahiy-stream/services/user-service/internal/domain"
	"github.com/sahiy/sahiy-stream/services/user-service/internal/usecase"
)

type UserServer struct {
	userv1.UnimplementedUserServiceServer
	uc *usecase.UserUseCase
}

func NewUserServer(uc *usecase.UserUseCase) *UserServer {
	return &UserServer{uc: uc}
}

func (s *UserServer) GetProfile(ctx context.Context, req *userv1.GetProfileRequest) (*userv1.Profile, error) {
	id, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, toGRPCError(err)
	}
	p, err := s.uc.GetProfile(ctx, id)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toProfile(p), nil
}

func (s *UserServer) UpdateProfile(ctx context.Context, req *userv1.UpdateProfileRequest) (*userv1.Profile, error) {
	id, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, toGRPCError(err)
	}
	p, err := s.uc.UpdateProfile(ctx, id, req.DisplayName, req.AvatarUrl)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toProfile(p), nil
}

func (s *UserServer) GetPublicProfile(ctx context.Context, req *userv1.GetPublicProfileRequest) (*userv1.PublicProfile, error) {
	p, err := s.uc.GetPublicProfile(ctx, req.GetUsername())
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &userv1.PublicProfile{
		Id:            p.ID.String(),
		Username:      p.Username,
		DisplayName:   p.DisplayName,
		CreatedAtUnix: p.CreatedAt.Unix(),
	}, nil
}

func toProfile(p *domain.Profile) *userv1.Profile {
	out := &userv1.Profile{
		Id:            p.ID.String(),
		Email:         p.Email,
		Username:      p.Username,
		DisplayName:   p.DisplayName,
		Role:          p.Role,
		EmailVerified: p.EmailVerified,
		CreatedAtUnix: p.CreatedAt.Unix(),
	}
	if p.AvatarURL != nil {
		out.AvatarUrl = *p.AvatarURL
	}
	return out
}
