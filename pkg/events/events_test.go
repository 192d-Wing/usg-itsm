package events_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/192d-Wing/usg-itsm/pkg/events"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// TestNATSPublisher_Publish verifies a published message lands in the stream.
// Runs only when TEST_NATS_URL points at a JetStream-enabled NATS.
func TestNATSPublisher_Publish(t *testing.T) {
	url := os.Getenv("TEST_NATS_URL")
	if url == "" {
		t.Skip("TEST_NATS_URL not set; skipping NATS integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	const stream = "ITSM_TEST"

	// Start from a clean stream so the count is deterministic.
	if nc, err := nats.Connect(url); err == nil {
		if js, err := jetstream.New(nc); err == nil {
			_ = js.DeleteStream(ctx, stream)
		}
		nc.Close()
	}

	pub, err := events.Connect(ctx, url, stream, []string{"itsmtest.>"})
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pub.Close()

	if err := pub.Publish(ctx, "itsmtest.ticket.created", []byte(`{"type":"ticket.created"}`)); err != nil {
		t.Fatalf("publish: %v", err)
	}

	// Inspect the stream via a separate client.
	nc, err := nats.Connect(url)
	if err != nil {
		t.Fatalf("inspect connect: %v", err)
	}
	defer nc.Close()
	js, err := jetstream.New(nc)
	if err != nil {
		t.Fatalf("jetstream: %v", err)
	}
	s, err := js.Stream(ctx, stream)
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	info, err := s.Info(ctx)
	if err != nil {
		t.Fatalf("info: %v", err)
	}
	if info.State.Msgs != 1 {
		t.Fatalf("want 1 message in stream, got %d", info.State.Msgs)
	}

	t.Cleanup(func() { _ = js.DeleteStream(ctx, stream) })
}
