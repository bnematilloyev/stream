package client

import (
	"context"

	"github.com/sahiy/sahiy-stream/pkg/grpcclient"
	authv1 "github.com/sahiy/sahiy-stream/proto/gen/auth/v1"
	"google.golang.org/grpc"
)

type AuthClient struct {
	conn   *grpc.ClientConn
	client authv1.AuthServiceClient
}

func NewAuthClient(addr string) (*AuthClient, error) {
	conn, err := grpcclient.Dial(addr, 15)
	if err != nil {
		return nil, err
	}

	return &AuthClient{
		conn:   conn,
		client: authv1.NewAuthServiceClient(conn),
	}, nil
}

func (c *AuthClient) Close() error {
	return c.conn.Close()
}

func (c *AuthClient) Conn() *grpc.ClientConn {
	return c.conn
}

func (c *AuthClient) Register(ctx context.Context, req *authv1.RegisterRequest) (*authv1.AuthResponse, error) {
	return c.client.Register(ctx, req)
}

func (c *AuthClient) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.AuthResponse, error) {
	return c.client.Login(ctx, req)
}

func (c *AuthClient) Refresh(ctx context.Context, req *authv1.RefreshRequest) (*authv1.AuthResponse, error) {
	return c.client.Refresh(ctx, req)
}

func (c *AuthClient) Logout(ctx context.Context, req *authv1.LogoutRequest) (*authv1.LogoutResponse, error) {
	return c.client.Logout(ctx, req)
}

func (c *AuthClient) ValidateToken(ctx context.Context, req *authv1.ValidateTokenRequest) (*authv1.ValidateTokenResponse, error) {
	return c.client.ValidateToken(ctx, req)
}

func (c *AuthClient) GetUser(ctx context.Context, req *authv1.GetUserRequest) (*authv1.User, error) {
	return c.client.GetUser(ctx, req)
}

func (c *AuthClient) ListUsers(ctx context.Context, req *authv1.ListUsersRequest) (*authv1.ListUsersResponse, error) {
	return c.client.ListUsers(ctx, req)
}

func (c *AuthClient) UpdateUserAdmin(ctx context.Context, req *authv1.UpdateUserAdminRequest) (*authv1.User, error) {
	return c.client.UpdateUserAdmin(ctx, req)
}

func (c *AuthClient) GetPlatformStats(ctx context.Context, req *authv1.GetPlatformStatsRequest) (*authv1.PlatformStatsResponse, error) {
	return c.client.GetPlatformStats(ctx, req)
}

func (c *AuthClient) ListAuditLogs(ctx context.Context, req *authv1.ListAuditLogsRequest) (*authv1.ListAuditLogsResponse, error) {
	return c.client.ListAuditLogs(ctx, req)
}
