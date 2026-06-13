package grpcclient

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func Dial(addr string, attempts int) (*grpc.ClientConn, error) {
	var conn *grpc.ClientConn
	var err error

	for i := 1; i <= attempts; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		conn, err = grpc.DialContext(ctx, addr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(),
		)
		cancel()
		if err == nil {
			return conn, nil
		}
		if i < attempts {
			time.Sleep(time.Second)
		}
	}
	return nil, fmt.Errorf("dial %s: %w", addr, err)
}
