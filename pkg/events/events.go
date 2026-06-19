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
	if _, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:     name,
		Subjects: subjects,
	}); err != nil {
		nc.Close()
		return nil, fmt.Errorf("ensure stream %q: %w", name, err)
	}
	return &NATSPublisher{nc: nc, js: js}, nil
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
