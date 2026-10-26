// Purpose: Keep only the latest pending stream snapshot between ordered worker notifications.
// Pattern: Bounded latest-value mailbox with explicit ordering barriers.
// Usage: A task worker owns delivery; receive consumes mailbox values on the terminal event loop.
// Related: internal/tui/session/events.go, internal/tui/session/app.go.
package session

import (
	"context"
	"sync"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

type streamNotice struct {
	mu      sync.Mutex
	event   runrecord.Event
	pending bool
}

func (s *streamNotice) take() *runrecord.Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.pending {
		return nil
	}
	event := s.event
	s.event = runrecord.Event{}
	s.pending = false
	return &event
}

type runtimeDelivery struct{ current *streamNotice }

func (d *runtimeDelivery) stream(ctx context.Context, queue chan<- notice, event runrecord.Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if d.current == nil {
		d.current = &streamNotice{}
	}
	current := d.current
	current.mu.Lock()
	pending := current.pending
	current.event, current.pending = event, true
	current.mu.Unlock()
	if pending {
		return nil
	}
	select {
	case queue <- notice{Stream: current}:
		return nil
	case <-ctx.Done():
		current.take()
		return ctx.Err()
	}
}

func (d *runtimeDelivery) post(ctx context.Context, queue chan<- notice, n notice) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	// Freeze earlier snapshots before retry, completion, usage or approval.
	// The next stream gets a distinct mailbox and cannot cross this barrier.
	d.current = nil
	select {
	case queue <- n:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
