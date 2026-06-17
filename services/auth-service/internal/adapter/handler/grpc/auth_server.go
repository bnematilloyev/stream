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

func (s *AuthServer) ListUsers(ctx context.Context, req *authv1.ListUsersRequest) (*authv1.ListUsersResponse, error) {
	users, total, err := s.uc.ListUsers(ctx, req.GetStatus(), req.GetRole(), req.GetSearch(), int(req.GetPage()), int(req.GetLimit()))
	if err != nil {
		return nil, toGRPCError(err)
	}
	out := make([]*authv1.User, 0, len(users))
	for i := range users {
		out = append(out, toProtoUser(&users[i]))
	}
	page, limit := int(req.GetPage()), int(req.GetLimit())
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	return &authv1.ListUsersResponse{Users: out, Total: int32(total), Page: int32(page), Limit: int32(limit)}, nil
}

func (s *AuthServer) UpdateUserAdmin(ctx context.Context, req *authv1.UpdateUserAdminRequest) (*authv1.User, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}
	var actorID uuid.UUID
	if req.GetActorId() != "" {
		actorID, err = uuid.Parse(req.GetActorId())
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid actor_id")
		}
	}
	var role, statusVal *string
	if req.Role != nil {
		v := req.GetRole()
		role = &v
	}
	if req.Status != nil {
		v := req.GetStatus()
		statusVal = &v
	}
	user, err := s.uc.UpdateUserAdmin(ctx, actorID, userID, role, statusVal)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toProtoUser(user), nil
}

func (s *AuthServer) GetPlatformStats(ctx context.Context, _ *authv1.GetPlatformStatsRequest) (*authv1.PlatformStatsResponse, error) {
	total, byStatus, admins, err := s.uc.GetPlatformStats(ctx)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &authv1.PlatformStatsResponse{
		TotalUsers:     total,
		UsersActive:    byStatus[domain.StatusActive],
		UsersSuspended: byStatus[domain.StatusSuspended],
		UsersBanned:    byStatus[domain.StatusBanned],
		Admins:         admins,
	}, nil
}

func (s *AuthServer) ListAuditLogs(ctx context.Context, req *authv1.ListAuditLogsRequest) (*authv1.ListAuditLogsResponse, error) {
	logs, total, err := s.uc.ListAuditLogs(ctx, int(req.GetPage()), int(req.GetLimit()))
	if err != nil {
		return nil, toGRPCError(err)
	}
	out := make([]*authv1.AuditLogEntry, 0, len(logs))
	for _, entry := range logs {
		item := &authv1.AuditLogEntry{
			Id: entry.ID, Action: entry.Action, ResourceType: entry.ResourceType,
			DetailsJson: entry.DetailsJSON, CreatedAtUnix: entry.CreatedAt.Unix(),
		}
		if entry.ActorID != nil {
			item.ActorId = entry.ActorID.String()
		}
		if entry.ResourceID != nil {
			item.ResourceId = entry.ResourceID.String()
		}
		out = append(out, item)
	}
	page, limit := int(req.GetPage()), int(req.GetLimit())
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	return &authv1.ListAuditLogsResponse{Logs: out, Total: int32(total), Page: int32(page), Limit: int32(limit)}, nil
}

func toAuthResponse(result *usecase.AuthResult) *authv1.AuthResponse {
	return &authv1.AuthResponse{
		User:          toProtoUser(result.User),
		AccessToken:   result.Tokens.AccessToken,
		RefreshToken:  result.Tokens.RefreshToken,
		ExpiresAtUnix: result.Tokens.ExpiresAt.Unix(),
	}
}

func toProtoUser(u *domain.User) *authv1.User {
	return &authv1.User{
		Id:            u.ID.String(),
		Email:         u.Email,
		Username:      u.Username,
		DisplayName:   u.DisplayName,
		Role:          u.Role,
		Status:        u.Status,
		EmailVerified: u.EmailVerified,
		CreatedAtUnix: u.CreatedAt.Unix(),
	}
}
