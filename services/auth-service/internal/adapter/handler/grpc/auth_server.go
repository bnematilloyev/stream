package grpc

import (
	"context"

	"github.com/google/uuid"
	authv1 "github.com/sahiy/sahiy-stream/proto/gen/auth/v1"
	"github.com/sahiy/sahiy-stream/services/auth-service/internal/domain"
	"github.com/sahiy/sahiy-stream/services/auth-service/internal/usecase"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuthServer struct {
	authv1.UnimplementedAuthServiceServer
	uc *usecase.AuthUseCase
}

func NewAuthServer(uc *usecase.AuthUseCase) *AuthServer {
	return &AuthServer{uc: uc}
}

func (s *AuthServer) Register(ctx context.Context, req *authv1.RegisterRequest) (*authv1.AuthResponse, error) {
	result, err := s.uc.Register(ctx, req.GetEmail(), req.GetUsername(), req.GetDisplayName(), req.GetPassword())
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toAuthResponse(result), nil
}

func (s *AuthServer) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.AuthResponse, error) {
	result, err := s.uc.Login(ctx, req.GetEmail(), req.GetPassword(), req.GetDeviceInfo(), req.GetIpAddress())
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toAuthResponse(result), nil
}

func (s *AuthServer) Refresh(ctx context.Context, req *authv1.RefreshRequest) (*authv1.AuthResponse, error) {
	result, err := s.uc.Refresh(ctx, req.GetRefreshToken())
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toAuthResponse(result), nil
}

func (s *AuthServer) Logout(ctx context.Context, req *authv1.LogoutRequest) (*authv1.LogoutResponse, error) {
	if err := s.uc.Logout(ctx, req.GetRefreshToken()); err != nil {
		return nil, toGRPCError(err)
	}
	return &authv1.LogoutResponse{Success: true}, nil
}

func (s *AuthServer) ValidateToken(ctx context.Context, req *authv1.ValidateTokenRequest) (*authv1.ValidateTokenResponse, error) {
	user, err := s.uc.ValidateAccess(ctx, req.GetAccessToken())
	if err != nil {
		return &authv1.ValidateTokenResponse{Valid: false}, nil
	}
	return &authv1.ValidateTokenResponse{
		Valid: true,
		User:  toProtoUser(user),
	}, nil
}

func (s *AuthServer) GetUser(ctx context.Context, req *authv1.GetUserRequest) (*authv1.User, error) {
	id, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}
	user, err := s.uc.GetUser(ctx, id)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toProtoUser(user), nil
}

func toAuthResponse(result *usecase.AuthResult) *authv1.AuthResponse {
	return &authv1.AuthResponse{
		User:            toProtoUser(result.User),
		AccessToken:     result.Tokens.AccessToken,
		RefreshToken:    result.Tokens.RefreshToken,
		ExpiresAtUnix:   result.Tokens.ExpiresAt.Unix(),
	}
}

func toProtoUser(u *domain.User) *authv1.User {
	return &authv1.User{
		Id:              u.ID.String(),
		Email:           u.Email,
		Username:        u.Username,
		DisplayName:     u.DisplayName,
		Role:            u.Role,
		Status:          u.Status,
		EmailVerified:   u.EmailVerified,
		CreatedAtUnix:   u.CreatedAt.Unix(),
	}
}
