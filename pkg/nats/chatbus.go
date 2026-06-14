package nats

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

// ChatBus publishes and consumes chat events via JetStream.
type ChatBus struct {
	nc *nats.Conn
	js nats.JetStreamContext
}

func NewChatBus(cfg Config) (*ChatBus, error) {
	nc, err := cfg.Connect()
	if err != nil {
		return nil, err
	}
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("jetstream: %w", err)
	}
	bus := &ChatBus{nc: nc, js: js}
	if err := bus.ensureStream(); err != nil {
		nc.Close()
		return nil, err
	}
	return bus, nil
}

func (b *ChatBus) ensureStream() error {
	_, err := b.js.StreamInfo(ChatStreamName)
	if err == nil {
		return nil
	}
	if err != nats.ErrStreamNotFound {
		return fmt.Errorf("stream info: %w", err)
	}
	_, err = b.js.AddStream(&nats.StreamConfig{
		Name:      ChatStreamName,
		Subjects:  []string{ChatSubjectPrefix + ">"},
		Retention: nats.LimitsPolicy,
		MaxAge:    24 * time.Hour,
		Storage:   nats.FileStorage,
	})
	if err != nil {
		return fmt.Errorf("add stream: %w", err)
	}
	return nil
}

func chatSubject(streamID string) string {
	return ChatSubjectPrefix + streamID
}

// Publish sends a chat event to all subscribers for the stream.
func (b *ChatBus) Publish(ctx context.Context, streamID string, payload []byte) error {
	_, err := b.js.PublishMsg(&nats.Msg{
		Subject: chatSubject(streamID),
		Data:    payload,
	}, nats.Context(ctx))
	if err != nil {
		return fmt.Errorf("publish chat: %w", err)
	}
	return nil
}

// Subscribe binds a queue group consumer for horizontal scaling.
func (b *ChatBus) Subscribe(handler func(streamID string, payload []byte)) (*nats.Subscription, error) {
	return b.js.QueueSubscribe(ChatSubjectAll, "chat-workers", func(msg *nats.Msg) {
		streamID := msg.Subject[len(ChatSubjectPrefix):]
		handler(streamID, msg.Data)
		_ = msg.Ack()
	}, nats.ManualAck())
}

// Ping verifies NATS connectivity.
func (b *ChatBus) Ping() error {
	return b.nc.Flush()
}

func (b *ChatBus) Close() {
	if b.nc != nil {
		b.nc.Close()
	}
}
