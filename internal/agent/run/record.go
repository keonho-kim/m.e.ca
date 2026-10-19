// Purpose: Persist each runtime event once with the owning request identifier.
// Pattern: Journal I/O boundary.
// Usage: Request transitions call record before publishing durable outcomes.
// Related: internal/agent/run/journal.go, internal/storage/sqlite/events.go.
package run

import (
	"context"
	"errors"
)

func (s *execution) record(ctx context.Context, event Event) error {
	event.RunID = s.checkpoint.ID
	if err := s.runner.Journal.RecordEvent(ctx, s.runner.Session, event); err != nil {
		return errors.Join(errRuntimeRecord, err)
	}
	return nil
}
