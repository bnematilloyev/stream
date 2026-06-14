package analytics

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// Config holds ClickHouse connection settings.
type Config struct {
	Addr     string
	Database string
	Username string
	Password string
	Enabled  bool
}

// Client writes analytics events to ClickHouse.
type Client struct {
	conn    driver.Conn
	enabled bool
}

// ViewerHeartbeatEvent records a live viewer heartbeat.
type ViewerHeartbeatEvent struct {
	StreamID   string
	SessionID  string
	Concurrent int64
	Unique     int64
	RecordedAt time.Time
}

func NewClient(cfg Config) (*Client, error) {
	if !cfg.Enabled {
		return &Client{enabled: false}, nil
	}
	if cfg.Database == "" {
		cfg.Database = "sahiy_analytics"
	}
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{cfg.Addr},
		Auth: clickhouse.Auth{
			Database: cfg.Database,
			Username: cfg.Username,
			Password: cfg.Password,
		},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("clickhouse open: %w", err)
	}
	if err := conn.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("clickhouse ping: %w", err)
	}
	return &Client{conn: conn, enabled: true}, nil
}

func (c *Client) Enabled() bool { return c.enabled }

func (c *Client) RecordViewerHeartbeat(ctx context.Context, evt ViewerHeartbeatEvent) error {
	if !c.enabled {
		return nil
	}
	if evt.RecordedAt.IsZero() {
		evt.RecordedAt = time.Now().UTC()
	}
	return c.conn.Exec(ctx, `
		INSERT INTO viewer_heartbeats (stream_id, session_id, concurrent, unique_count, recorded_at)
		VALUES (?, ?, ?, ?, ?)
	`, evt.StreamID, evt.SessionID, evt.Concurrent, evt.Unique, evt.RecordedAt)
}

func (c *Client) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}
