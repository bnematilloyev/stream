package health

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

// PostgresChecker pings a PostgreSQL connection pool.
type PostgresChecker struct {
	Pool *pgxpool.Pool
}

func (c PostgresChecker) Name() string { return "postgres" }

func (c PostgresChecker) Check(ctx context.Context) error {
	return c.Pool.Ping(ctx)
}

// RedisChecker pings a Redis client.
type RedisChecker struct {
	Client *redis.Client
}

func (c RedisChecker) Name() string { return "redis" }

func (c RedisChecker) Check(ctx context.Context) error {
	return c.Client.Ping(ctx).Err()
}

// GRPCChecker verifies a gRPC connection is ready.
type GRPCChecker struct {
	ServiceName string
	Conn        *grpc.ClientConn
}

func (c GRPCChecker) Check(ctx context.Context) error {
	state := c.Conn.GetState()
	if state == connectivity.Ready || state == connectivity.Idle {
		return nil
	}
	if !c.Conn.WaitForStateChange(ctx, state) {
		return context.DeadlineExceeded
	}
	if c.Conn.GetState() == connectivity.Ready || c.Conn.GetState() == connectivity.Idle {
		return nil
	}
	return context.DeadlineExceeded
}

func (c GRPCChecker) Name() string {
	if c.ServiceName != "" {
		return c.ServiceName
	}
	return "grpc"
}
