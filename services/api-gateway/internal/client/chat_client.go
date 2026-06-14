package client

import (
	"github.com/sahiy/sahiy-stream/pkg/grpcclient"
	chatv1 "github.com/sahiy/sahiy-stream/proto/gen/chat/v1"
	"google.golang.org/grpc"
)

type ChatClient struct {
	conn *grpc.ClientConn
	Chat chatv1.ChatServiceClient
}

func NewChatClient(addr string) (*ChatClient, error) {
	conn, err := grpcclient.Dial(addr, 10)
	if err != nil {
		return nil, err
	}
	return &ChatClient{
		conn: conn,
		Chat: chatv1.NewChatServiceClient(conn),
	}, nil
}

func (c *ChatClient) Close() error { return c.conn.Close() }

func (c *ChatClient) Conn() *grpc.ClientConn { return c.conn }
