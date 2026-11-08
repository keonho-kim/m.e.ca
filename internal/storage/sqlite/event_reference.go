// Purpose: Return the latest committed journal event without querying again after each live result.
// Pattern: Commit-owned atomic receipt projection.
// Usage: The native runtime attaches this reference to terminal presentation events.
// Related: internal/storage/sqlite/commit.go, internal/agent/run/start.go.
package sqlite

import (
	"context"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func (s *Store) LatestEventReference(ctx context.Context) (*runrecord.EventReference, error) {
	ref := &runrecord.EventReference{Session: s.session.ID}
	ref.Sequence = s.latestEvent.Load()
	if ref.Sequence > 0 {
		return ref, nil
	}
	var seq int64
	err := s.db.QueryRowContext(ctx, "SELECT COALESCE(MAX(sequence),0) FROM tool_events").Scan(&seq)
	if err != nil {
		return nil, err
	}
	s.latestEvent.CompareAndSwap(0, seq)
	ref.Sequence = s.latestEvent.Load()
	if ref.Sequence < 1 {
		return nil, faults.Errorf(diagcatalog.SqliteNoCommittedEventIsAvailable)
	}
	return ref, nil
}
