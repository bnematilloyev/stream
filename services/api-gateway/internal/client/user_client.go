package client

import (
	"github.com/sahiy/sahiy-stream/pkg/grpcclient"
	userv1 "github.com/sahiy/sahiy-stream/proto/gen/user/v1"
	"google.golang.org/grpc"
)

type UserClient struct {
	conn      *grpc.ClientConn
	User      userv1.UserServiceClient
	Channel   userv1.ChannelServiceClient
}

func NewUserClient(addr string) (*UserClient, error) {
	conn, err := grpcclient.Dial(addr, 15)
	if err != nil {
		return nil, err
	}
	return &UserClient{
		conn:    conn,
		User:    userv1.NewUserServiceClient(conn),
		Channel: userv1.NewChannelServiceClient(conn),
	}, nil
}

func (c *UserClient) Close() error { return c.conn.Close() }
