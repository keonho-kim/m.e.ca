// Purpose: Verify stream snapshots do not crowd out ordered terminal notifications.
// Pattern: Deterministic event delivery regression tests.
// Usage: go test -race ./internal/tui/session -run TestStreamDelivery.
// Related: internal/tui/session/events.go, internal/tui/session/runtime_events.go.
package session

import (
	"context"
	"fmt"
	"sync"
	"testing"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

func TestStreamDeliveryKeepsLatestSnapshot(t *testing.T) {
	a := &app{events: make(chan notice, 64)}
	delivery := runtimeDelivery{}
	for i := range 32 {
		if err := delivery.stream(context.Background(), a.events, runrecord.Event{Kind: "stream", Text: fmt.Sprint(i)}); err != nil {
			t.Fatal(err)
		}
	}
	if len(a.events) != 1 {
		t.Fatalf("queued %d obsolete snapshots; want one latest value", len(a.events))
	}
}

func TestStreamDeliveryPreservesBoundariesAndReleasesSnapshots(t *testing.T) {
	a := &app{events: make(chan notice, 64)}
	delivery := runtimeDelivery{}
	ctx := context.Background()
	for _, kind := range []string{"stream", "stream", "retry", "stream", "usage", "thinking", "response"} {
		event := &runrecord.Event{Kind: kind, Text: kind}
		post := func() error {
			if kind == "stream" {
				return delivery.stream(ctx, a.events, *event)
			}
			return delivery.post(ctx, a.events, notice{Event: event})
		}
		if err := post(); err != nil {
			t.Fatal(err)
		}
	}
	for _, kind := range []string{"stream", "retry", "stream", "usage", "thinking", "response"} {
		n := <-a.events
		event := n.Event
		if n.Stream != nil {
			event = n.Stream.take()
			if n.Stream.take() != nil {
				t.Fatal("consumed snapshot retained")
			}
		}
		if event == nil || event.Kind != kind {
			t.Fatalf("want %s; got %+v", kind, event)
		}
	}
	ctx, cancel := context.WithCancel(ctx)
	cancel()
	if err := delivery.stream(ctx, a.events, runrecord.Event{Kind: "stream"}); err == nil || len(a.events) != 0 {
		t.Fatal("cancelled delivery retained a snapshot")
	}
}

func TestStreamDeliveryConcurrentConsumption(t *testing.T) {
	a := &app{events: make(chan notice, 64)}
	delivery := runtimeDelivery{}
	var consumer sync.WaitGroup
	consumer.Go(func() {
		for {
			n := <-a.events
			if n.Done {
				return
			}
			if n.Stream != nil {
				n.Stream.take()
			}
		}
	})
	for range 1000 {
		if err := delivery.stream(context.Background(), a.events, runrecord.Event{Kind: "stream", Thinking: "snapshot"}); err != nil {
			t.Fatal(err)
		}
	}
	delivery.post(context.Background(), a.events, notice{Done: true})
	consumer.Wait()
}
