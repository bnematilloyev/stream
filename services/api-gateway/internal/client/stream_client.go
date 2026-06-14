package client

import (
	"github.com/sahiy/sahiy-stream/pkg/grpcclient"
	streamv1 "github.com/sahiy/sahiy-stream/proto/gen/stream/v1"
	"google.golang.org/grpc"
)

type StreamClient struct {
	conn   *grpc.ClientConn
	Stream streamv1.StreamServiceClient
}

func NewStreamClient(addr string) (*StreamClient, error) {
	conn, err := grpcclient.Dial(addr, 15)
	if err != nil {
		return nil, err
	}
	return &StreamClient{conn: conn, Stream: streamv1.NewStreamServiceClient(conn)}, nil
}

func (c *StreamClient) Close() error { return c.conn.Close() }

func (c *StreamClient) Conn() *grpc.ClientConn { return c.conn }
