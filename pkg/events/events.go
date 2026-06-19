// Package events publishes domain events to NATS JetStream (ADR-0004). The
// database remains each service's durable record; the bus is for fan-out to
// other services (notifications, audit, search projections).
package events

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Publisher publishes a message to a subject.
type Publisher interface {
	Publish(ctx context.Context, subject string, data []byte) error
	Close()
}

// NopPublisher discards events; used when NATS is not configured or in tests.
type NopPublisher struct{}

func (NopPublisher) Publish(context.Context, string, []byte) error { return nil }
func (NopPublisher) Close()                                        {}

// NATSPublisher publishes to a JetStream stream.
type NATSPublisher struct {
	nc *nats.Conn
	js jetstream.JetStream
}

// Connect dials NATS, initializes JetStream, and ensures a stream named name
// exists covering subjects (e.g. []string{"itsm.>"}).
func Connect(ctx context.Context, url, name string, subjects []string) (*NATSPublisher, error) {
	nc, err := nats.Connect(url, nats.Name("usg-itsm"))
	if err != nil {
		return nil, fmt.Errorf("nats connect: %w", err)
	}
	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("jetstream init: %w", err)
	}
	if err := ensureStream(ctx, js, name, subjects); err != nil {
		nc.Close()
		return nil, err
	}
	return &NATSPublisher{nc: nc, js: js}, nil
}

// ensureStream idempotently creates/updates a JetStream stream. Publisher and
// consumers call it with the same name/subjects, so order of startup doesn't
// matter.
func ensureStream(ctx context.Context, js jetstream.JetStream, name string, subjects []string) error {
	if _, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:     name,
		Subjects: subjects,
	}); err != nil {
		return fmt.Errorf("ensure stream %q: %w", name, err)
	}
	return nil
}

// Publish queues data to subject without blocking on the JetStream ack
// (PublishAsync). This keeps callers off the network hot path and decouples
// delivery from the request context: the NATS client sends from its own
// background goroutine and buffers across reconnects. ctx is unused here but
// kept for the Publisher contract. Delivery is best-effort; flush on Close.
func (p *NATSPublisher) Publish(_ context.Context, subject string, data []byte) error {
	if _, err := p.js.PublishAsync(subject, data); err != nil {
		return fmt.Errorf("publish %s: %w", subject, err)
	}
	return nil
}

// Close flushes pending async publishes (bounded) then drains the connection.
func (p *NATSPublisher) Close() {
	select {
	case <-p.js.PublishAsyncComplete():
	case <-time.After(3 * time.Second):
	}
	_ = p.nc.Drain()
}

// Handler processes one event. Returning an error nak's the message so
// JetStream redelivers it (up to the consumer's MaxDeliver).
type Handler func(subject string, data []byte) error

// ConsumeConfig configures a durable consumer.
type ConsumeConfig struct {
	Stream     string   // stream name, e.g. "ITSM"
	Durable    string   // durable consumer name (stable across restarts)
	Subjects   []string // filter subjects, e.g. []string{"itsm.ticket.*"}
	MaxDeliver int      // redelivery cap before a poison message is dropped (default 5)
}

// Consumer is a running durable JetStream subscription.
type Consumer struct {
	nc *nats.Conn
	cc jetstream.ConsumeContext
}

// Consume binds a durable consumer on cfg.Stream and dispatches matching
// messages to handler, acking on success and nak'ing on error (redelivered up
// to MaxDeliver). The stream is ensured so the consumer can start before the
// publisher.
func Consume(ctx context.Context, url string, cfg ConsumeConfig, handler Handler) (*Consumer, error) {
	if cfg.MaxDeliver <= 0 {
		cfg.MaxDeliver = 5
	}
	nc, err := nats.Connect(url, nats.Name("usg-itsm"))
	if err != nil {
		return nil, fmt.Errorf("nats connect: %w", err)
	}
	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("jetstream init: %w", err)
	}
	if err := ensureStream(ctx, js, cfg.Stream, []string{"itsm.>"}); err != nil {
		nc.Close()
		return nil, err
	}

	cons, err := js.CreateOrUpdateConsumer(ctx, cfg.Stream, jetstream.ConsumerConfig{
		Durable:        cfg.Durable,
		FilterSubjects: cfg.Subjects,
		AckPolicy:      jetstream.AckExplicitPolicy,
		MaxDeliver:     cfg.MaxDeliver,
	})
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("consumer: %w", err)
	}

	cc, err := cons.Consume(func(msg jetstream.Msg) {
		if err := handler(msg.Subject(), msg.Data()); err != nil {
			_ = msg.Nak()
			return
		}
		_ = msg.Ack()
	})
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("consume: %w", err)
	}
	return &Consumer{nc: nc, cc: cc}, nil
}

// Close stops consuming and drains the connection.
func (c *Consumer) Close() {
	c.cc.Stop()
	_ = c.nc.Drain()
}
