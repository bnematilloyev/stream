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
	conn, err := grpcclient.Dial(addr, 10)
	if err != nil {
		return nil, err
	}
	return &AuthClient{
		conn:   conn,
		client: authv1.NewAuthServiceClient(conn),
	}, nil
}

func (c *AuthClient) Close() error { return c.conn.Close() }

func (c *AuthClient) GetUser(ctx context.Context, req *authv1.GetUserRequest) (*authv1.User, error) {
	return c.client.GetUser(ctx, req)
}
