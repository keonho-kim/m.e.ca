// Purpose: Await compatible task completion without copying unrelated progress events.
// Pattern: Process-owned task completion signal.
// Usage: Nonstreaming and buffered OpenAI observers wait here with their own cancellation context.
// Related: internal/commands/server/execution.go, internal/commands/server/events.go.
package server

import (
	"context"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func (s *Service) WaitResponse(ctx context.Context, id string) (Response, error) {
	for {
		// Register before reading status, so completion or a failed admission
		// cannot be missed between a read and a wait.
		s.mu.Lock()
		var changed chan struct{}
		if r := s.active; r != nil && r.response != nil && r.response.ID == id {
			changed = r.done
		} else {
			if s.feed.changed == nil {
				s.feed.changed = make(chan struct{})
			}
			changed = s.feed.changed
		}
		s.mu.Unlock()
		value, err := s.ResponseState(ctx, id, false)
		if err != nil || value.Status != "queued" && value.Status != "in_progress" {
			return value, err
		}
		select {
		case <-ctx.Done():
			return Response{}, ctx.Err()
		case <-s.ctx.Done():
			return Response{}, faults.New(diagcatalog.Unavailable, diagcatalog.ServerServerIsShuttingDown)
		case <-changed:
		}
	}
}
