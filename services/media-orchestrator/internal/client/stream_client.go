package client

import (
	"context"

	"github.com/sahiy/sahiy-stream/pkg/grpcclient"
	streamv1 "github.com/sahiy/sahiy-stream/proto/gen/stream/v1"
	"google.golang.org/grpc"
)

type StreamClient struct {
	conn *grpc.ClientConn
	api  streamv1.StreamServiceClient
}

func NewStreamClient(addr string) (*StreamClient, error) {
	conn, err := grpcclient.Dial(addr, 15)
	if err != nil {
		return nil, err
	}
	return &StreamClient{conn: conn, api: streamv1.NewStreamServiceClient(conn)}, nil
}

func (c *StreamClient) Close() error { return c.conn.Close() }

func (c *StreamClient) ValidateStreamKey(ctx context.Context, key string) (bool, string, string, error) {
	res, err := c.api.ValidateStreamKey(ctx, &streamv1.ValidateStreamKeyRequest{StreamKey: key})
	if err != nil {
		return false, "", "", err
	}
	return res.Valid, res.ChannelId, res.StreamId, nil
}

func (c *StreamClient) StartIngest(ctx context.Context, channelID string) (string, error) {
	st, err := c.api.StartIngest(ctx, &streamv1.StartIngestRequest{ChannelId: channelID})
	if err != nil {
		return "", err
	}
	return st.Id, nil
}

func (c *StreamClient) EndIngest(ctx context.Context, streamID string) error {
	_, err := c.api.EndIngest(ctx, &streamv1.EndIngestRequest{StreamId: streamID})
	return err
}

func (c *StreamClient) GetStream(ctx context.Context, streamID string) (latencyMode, status string, err error) {
	st, err := c.api.GetStream(ctx, &streamv1.GetStreamRequest{StreamId: streamID})
	if err != nil {
		return "", "", err
	}
	return st.LatencyMode, st.Status, nil
}
