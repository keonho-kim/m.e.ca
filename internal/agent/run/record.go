// Purpose: Persist each runtime event once with the owning request identifier.
// Pattern: Journal I/O boundary.
// Usage: Request transitions call record before publishing durable outcomes.
// Related: internal/agent/run/journal.go, internal/storage/sqlite/events.go.
package run

import (
	"context"
	"errors"
	"time"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

func (s *execution) record(ctx context.Context, event runrecord.Event) error {
	return s.recordEvents(ctx, []runrecord.Event{event})
}

func (s *execution) recordEvents(ctx context.Context, events []runrecord.Event) error {
	records := make([]runrecord.Entry, len(events))
	for i := range events {
		events[i].RunID = s.checkpoint.ID
		if events[i].OccurredAt == "" {
			events[i].OccurredAt = time.Now().UTC().Format(time.RFC3339Nano)
		}
		records[i].Event = &events[i]
	}
	if err := s.runner.Journal.Commit(ctx, s.runner.Session, records); err != nil {
		return errors.Join(errRuntimeRecord, err)
	}
	return nil
}
