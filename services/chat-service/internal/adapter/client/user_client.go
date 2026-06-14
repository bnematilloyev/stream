package client

import (
	"context"

	"github.com/sahiy/sahiy-stream/pkg/grpcclient"
	userv1 "github.com/sahiy/sahiy-stream/proto/gen/user/v1"
	"google.golang.org/grpc"
)

type UserClient struct {
	conn   *grpc.ClientConn
	client userv1.ChannelServiceClient
}

func NewUserClient(addr string) (*UserClient, error) {
	conn, err := grpcclient.Dial(addr, 10)
	if err != nil {
		return nil, err
	}
	return &UserClient{
		conn:   conn,
		client: userv1.NewChannelServiceClient(conn),
	}, nil
}

func (c *UserClient) Close() error { return c.conn.Close() }

func (c *UserClient) GetChannel(ctx context.Context, slug string) (*userv1.Channel, error) {
	return c.client.GetChannel(ctx, &userv1.GetChannelRequest{Slug: slug})
}
