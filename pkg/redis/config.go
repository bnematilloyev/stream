package redis

import (
	"context"
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"
)

// Config supports standalone and cluster Redis topologies.
type Config struct {
	URL          string
	ClusterAddrs []string
	PoolSize     int
}

// NewClientFromConfig returns a universal Redis client (standalone or cluster).
func NewClientFromConfig(cfg Config) (redis.UniversalClient, error) {
	if len(cfg.ClusterAddrs) > 0 {
		client := redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:    cfg.ClusterAddrs,
			PoolSize: poolSize(cfg.PoolSize),
		})
		if err := client.Ping(context.Background()).Err(); err != nil {
			return nil, fmt.Errorf("ping redis cluster: %w", err)
		}
		return client, nil
	}

	opts, err := redis.ParseURL(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	if cfg.PoolSize > 0 {
		opts.PoolSize = cfg.PoolSize
	}
	client := redis.NewClient(opts)
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}
	return client, nil
}

// ParseClusterAddrs splits comma-separated cluster node addresses.
func ParseClusterAddrs(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if addr := strings.TrimSpace(p); addr != "" {
			out = append(out, addr)
		}
	}
	return out
}

func poolSize(size int) int {
	if size <= 0 {
		return 20
	}
	return size
}
