// Purpose: Read shared replay/terminal facts and snapshot the selected workspace independently.
// Pattern: Cancellable replay reader.
// Usage: SSE, socket and RPC observers call NextTask and ReplayTask.
// Related: internal/commands/server/events.go, internal/commands/server/snapshot.go.
package server

import (
	"context"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"strconv"
	"strings"
)

func (f *eventLog) outcome(workspace, id string) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	for index := f.count - 1; index >= 0; index-- {
		value := f.entry(index).value
		if value.Workspace == workspace && value.Task == id && value.Outcome != nil && value.Record.Kind == "request_finished" {
			return string(value.Outcome.Status)
		}
	}
	return ""
}

func (s *Service) Next(ctx context.Context, cursor string) (Batch, error) {
	return s.NextTask(ctx, cursor, "")
}

func (s *Service) NextTask(ctx context.Context, cursor, taskID string) (Batch, error) {
	for {
		s.mu.Lock()
		batch, ready, err := s.readEvents(cursor, taskID)
		if err != nil || ready {
			s.mu.Unlock()
			return batch, err
		}
		if s.closed {
			s.mu.Unlock()
			return Batch{}, faults.New(diagcatalog.Unavailable, diagcatalog.ServerServerIsShuttingDown)
		}
		s.feed.mu.Lock()
		if cursor != s.feed.cursorLocked() {
			s.feed.mu.Unlock()
			s.mu.Unlock()
			continue
		}
		if s.feed.changed == nil {
			s.feed.changed = make(chan struct{})
		}
		changed := s.feed.changed
		s.feed.mu.Unlock()
		s.mu.Unlock()
		select {
		case <-ctx.Done():
			return Batch{}, ctx.Err()
		case <-changed:
		}
	}
}

// Replay validates a cursor and reads available events without waiting. SSE
// uses it to establish response headers even when the subscriber is caught up.
func (s *Service) Replay(cursor string) (Batch, error) { return s.ReplayTask(cursor, "") }

func (s *Service) ReplayTask(cursor, taskID string) (Batch, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	batch, _, err := s.readEvents(cursor, taskID)
	return batch, err
}

func (s *Service) readEvents(cursor, taskID string) (Batch, bool, error) {
	f := s.feed
	f.mu.Lock()
	locked := true
	defer func() {
		if locked {
			f.mu.Unlock()
		}
	}()
	reset := func() (Batch, bool, error) {
		f.mu.Unlock()
		locked = false
		value := s.snapshotPage(taskID, "", 32)
		return Batch{Snapshot: &value}, true, nil
	}
	if cursor == "" {
		return reset()
	}
	epoch, raw, ok := strings.Cut(cursor, ":")
	sequence, err := strconv.ParseUint(raw, 10, 64)
	if !ok || err != nil || len(epoch) != 26 {
		return Batch{}, false, faults.New(diagcatalog.InvalidArgument, diagcatalog.ServerInvalidEventCursor)
	}
	if epoch != f.epoch || (f.count > 0 && sequence < f.entry(0).sequence-1) {
		return reset()
	}
	if sequence > f.sequence {
		return Batch{}, false, faults.New(diagcatalog.InvalidArgument, diagcatalog.ServerEventCursorIsAheadOfThisServer)
	}
	if sequence == f.sequence {
		return Batch{}, false, nil
	}
	// Retained sequences are contiguous, so a cursor locates its next entry
	// directly regardless of how much older history remains in the log.
	start := int(sequence + 1 - f.entry(0).sequence)
	end := min(start+32, f.count)
	values := make([]Event, end-start)
	for i := range values {
		values[i] = f.entry(start + i).value
	}
	return Batch{Events: values}, len(values) > 0, nil
}
