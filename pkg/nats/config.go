package nats

import (
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

const (
	ChatStreamName   = "CHAT"
	ChatSubjectPrefix = "chat."
	ChatSubjectAll   = "chat.>"
)

// Config holds NATS connection settings.
type Config struct {
	URL            string
	ConnectTimeout time.Duration
}

func DefaultConfig(url string) Config {
	if url == "" {
		url = nats.DefaultURL
	}
	return Config{URL: url, ConnectTimeout: 5 * time.Second}
}

func (c Config) Connect() (*nats.Conn, error) {
	opts := []nats.Option{
		nats.Name("sahiy-stream"),
		nats.Timeout(c.ConnectTimeout),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(time.Second),
	}
	nc, err := nats.Connect(c.URL, opts...)
	if err != nil {
		return nil, fmt.Errorf("nats connect: %w", err)
	}
	return nc, nil
}
