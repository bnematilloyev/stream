package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/sahiy/sahiy-stream/pkg/transcode"
)

const (
	TranscodeCmdStream = "TRANSCODE_CMD"
	TranscodeEvtStream = "TRANSCODE_EVT"
	TranscodeWorkersQ  = "transcode-workers"
)

// TranscodeBus routes transcode commands and events via JetStream.
type TranscodeBus struct {
	nc *nats.Conn
	js nats.JetStreamContext
}

func NewTranscodeBus(cfg Config) (*TranscodeBus, error) {
	nc, err := cfg.Connect()
	if err != nil {
		return nil, err
	}
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("jetstream: %w", err)
	}
	bus := &TranscodeBus{nc: nc, js: js}
	if err := bus.ensureStreams(); err != nil {
		nc.Close()
		return nil, err
	}
	return bus, nil
}

func (b *TranscodeBus) ensureStreams() error {
	if err := b.ensureStream(TranscodeCmdStream, []string{
		transcode.CmdStartSubject,
		transcode.CmdStopSubject,
	}, nats.WorkQueuePolicy, 0); err != nil {
		return err
	}
	return b.ensureStream(TranscodeEvtStream, []string{
		transcode.EvtSubjectPrefix + ">",
	}, nats.LimitsPolicy, 24*time.Hour)
}

func (b *TranscodeBus) ensureStream(name string, subjects []string, retention nats.RetentionPolicy, maxAge time.Duration) error {
	_, err := b.js.StreamInfo(name)
	if err == nil {
		return nil
	}
	if err != nats.ErrStreamNotFound {
		return fmt.Errorf("stream info %s: %w", name, err)
	}
	cfg := &nats.StreamConfig{
		Name:      name,
		Subjects:  subjects,
		Retention: retention,
		Storage:   nats.FileStorage,
	}
	if maxAge > 0 {
		cfg.MaxAge = maxAge
	}
	_, err = b.js.AddStream(cfg)
	if err != nil {
		return fmt.Errorf("add stream %s: %w", name, err)
	}
	return nil
}

func (b *TranscodeBus) PublishStart(ctx context.Context, job transcode.StartJob) error {
	return b.publishJSON(ctx, transcode.CmdStartSubject, job)
}

func (b *TranscodeBus) PublishStop(ctx context.Context, job transcode.StopJob) error {
	return b.publishJSON(ctx, transcode.CmdStopSubject, job)
}

func (b *TranscodeBus) PublishEvent(ctx context.Context, evt transcode.JobEvent) error {
	return b.publishJSON(ctx, transcode.EvtSubject(evt.StreamID), evt)
}

func (b *TranscodeBus) publishJSON(ctx context.Context, subject string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	_, err = b.js.PublishMsg(&nats.Msg{Subject: subject, Data: data}, nats.Context(ctx))
	if err != nil {
		return fmt.Errorf("publish %s: %w", subject, err)
	}
	return nil
}

// CommandHandler receives start/stop commands.
type CommandHandler func(subject string, data []byte) error

// SubscribeCommands binds a queue group consumer for start/stop transcode commands.
// A single wildcard subscription avoids JetStream "subject does not match consumer"
// when start and stop share the same queue group on one stream.
func (b *TranscodeBus) SubscribeCommands(handler CommandHandler) (*nats.Subscription, error) {
	return b.js.QueueSubscribe(
		transcode.CmdAllSubject,
		TranscodeWorkersQ,
		func(msg *nats.Msg) { b.handleCommand(msg, handler) },
		nats.ManualAck(),
		nats.BindStream(TranscodeCmdStream),
	)
}

// SubscribeStopCommands is deprecated; use SubscribeCommands (wildcard covers stop).
func (b *TranscodeBus) SubscribeStopCommands(handler CommandHandler) (*nats.Subscription, error) {
	return b.SubscribeCommands(handler)
}

func (b *TranscodeBus) handleCommand(msg *nats.Msg, handler CommandHandler) {
	if err := handler(msg.Subject, msg.Data); err != nil {
		_ = msg.Nak()
		return
	}
	_ = msg.Ack()
}

// EventHandler receives worker lifecycle events.
type EventHandler func(evt transcode.JobEvent) error

// SubscribeEvents listens for all transcode worker events.
func (b *TranscodeBus) SubscribeEvents(handler EventHandler) (*nats.Subscription, error) {
	return b.js.Subscribe(transcode.EvtSubjectPrefix+">", func(msg *nats.Msg) {
		var evt transcode.JobEvent
		if err := json.Unmarshal(msg.Data, &evt); err != nil {
			_ = msg.Nak()
			return
		}
		if err := handler(evt); err != nil {
			_ = msg.Nak()
			return
		}
		_ = msg.Ack()
	}, nats.Durable("transcode-orchestrator"), nats.ManualAck(), nats.BindStream(TranscodeEvtStream))
}

func (b *TranscodeBus) Ping() error { return b.nc.Flush() }

func (b *TranscodeBus) Close() {
	if b.nc != nil {
		b.nc.Close()
	}
}
