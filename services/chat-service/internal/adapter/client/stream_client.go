package client

import (
	"context"

	"github.com/google/uuid"
	"github.com/sahiy/sahiy-stream/pkg/grpcclient"
	streamv1 "github.com/sahiy/sahiy-stream/proto/gen/stream/v1"
	"github.com/sahiy/sahiy-stream/services/chat-service/internal/domain"
	"google.golang.org/grpc"
)

type StreamClient struct {
	client streamv1.StreamServiceClient
	conn   *grpc.ClientConn
}

func NewStreamClient(addr string) (*StreamClient, error) {
	conn, err := grpcclient.Dial(addr, 10)
	if err != nil {
		return nil, err
	}
	return &StreamClient{
		client: streamv1.NewStreamServiceClient(conn),
		conn:   conn,
	}, nil
}

func (c *StreamClient) Close() error { return c.conn.Close() }

func (c *StreamClient) IsLive(ctx context.Context, streamID uuid.UUID) (bool, error) {
	st, err := c.GetStream(ctx, streamID)
	if err != nil {
		return false, err
	}
	return st.GetStatus() == "live", nil
}

func (c *StreamClient) GetStream(ctx context.Context, streamID uuid.UUID) (*streamv1.Stream, error) {
	return c.client.GetStream(ctx, &streamv1.GetStreamRequest{StreamId: streamID.String()})
}

var _ domain.StreamChecker = (*StreamClient)(nil)
