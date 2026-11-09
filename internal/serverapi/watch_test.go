// Purpose: Verify that canceled subscriptions stop before sending another retained notification.
// Pattern: Deterministic subscription lifecycle tests.
// Usage: go test ./internal/serverapi -run TestWatchCancellation.
// Related: internal/serverapi/watch.go, internal/commands/server/events.go.
package serverapi

import (
	"context"
	"errors"
	"testing"

	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
)

func TestWatchCancellationStopsRetainedBatch(t *testing.T) {
	f := newFixture(t, generateFunc(idleGenerator), false)
	state, err := f.app.Select(deadline(t), "", true)
	if err != nil {
		t.Fatal(err)
	}
	permission := state.Permission
	for range 3 {
		if _, err := f.app.Configure(deadline(t), f.app.State().Session.ID, &permission, nil); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithCancel(deadline(t))
	defer cancel()
	sent := 0
	err = f.api.watch(ctx, state.Cursor, "", func(*v1.Notification) error {
		sent++
		cancel()
		return nil
	})
	if !errors.Is(err, context.Canceled) || sent != 1 {
		t.Fatalf("canceled subscription sent %d notifications; error: %v", sent, err)
	}
}

func TestWatchCancellationPrecedesSnapshot(t *testing.T) {
	f := newFixture(t, generateFunc(idleGenerator), false)
	ctx, cancel := context.WithCancel(deadline(t))
	cancel()
	sent := 0
	err := f.api.watch(ctx, "", "", func(*v1.Notification) error {
		sent++
		return context.Canceled
	})
	if !errors.Is(err, context.Canceled) || sent != 0 {
		t.Fatalf("canceled subscription sent %d snapshots; error: %v", sent, err)
	}
}
