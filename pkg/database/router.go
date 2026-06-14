package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Router routes queries to primary (write) or replica (read) pools.
type Router struct {
	primary *pgxpool.Pool
	replica *pgxpool.Pool
}

// PoolsConfig holds primary and optional read-replica URLs.
type PoolsConfig struct {
	PrimaryURL string
	ReplicaURL string
	Pool       Config
}

// NewRouter creates primary pool and optional replica pool.
func NewRouter(ctx context.Context, cfg PoolsConfig) (*Router, error) {
	if cfg.Pool.URL == "" {
		cfg.Pool = DefaultConfig(cfg.PrimaryURL)
	} else if cfg.Pool.URL != cfg.PrimaryURL && cfg.PrimaryURL != "" {
		cfg.Pool.URL = cfg.PrimaryURL
	}

	primary, err := NewPool(ctx, cfg.Pool)
	if err != nil {
		return nil, fmt.Errorf("primary pool: %w", err)
	}

	router := &Router{primary: primary}
	if cfg.ReplicaURL != "" && cfg.ReplicaURL != cfg.PrimaryURL {
		replicaCfg := cfg.Pool
		replicaCfg.URL = cfg.ReplicaURL
		replica, err := NewPool(ctx, replicaCfg)
		if err != nil {
			primary.Close()
			return nil, fmt.Errorf("replica pool: %w", err)
		}
		router.replica = replica
	}
	return router, nil
}

// Write returns the primary pool for mutations.
func (r *Router) Write() *pgxpool.Pool { return r.primary }

// Read returns the replica pool, falling back to primary.
func (r *Router) Read() *pgxpool.Pool {
	if r.replica != nil {
		return r.replica
	}
	return r.primary
}

// Primary returns the write pool (alias for health checks).
func (r *Router) Primary() *pgxpool.Pool { return r.primary }

// Replica returns the read pool or nil when not configured.
func (r *Router) Replica() *pgxpool.Pool { return r.replica }

// HasReplica reports whether a dedicated read pool is configured.
func (r *Router) HasReplica() bool { return r.replica != nil }

// Close closes all pools.
func (r *Router) Close() {
	r.primary.Close()
	if r.replica != nil {
		r.replica.Close()
	}
}

// LoadPoolsConfigFromEnv builds pool config from standard env vars.
func LoadPoolsConfigFromEnv(primaryURL, replicaURL string, maxConns, minConns int32) PoolsConfig {
	pool := DefaultConfig(primaryURL)
	if maxConns > 0 {
		pool.MaxConns = maxConns
	}
	if minConns > 0 {
		pool.MinConns = minConns
	}
	pool.MaxConnLifetime = time.Hour
	pool.MaxConnIdleTime = 30 * time.Minute
	return PoolsConfig{
		PrimaryURL: primaryURL,
		ReplicaURL: replicaURL,
		Pool:       pool,
	}
}
