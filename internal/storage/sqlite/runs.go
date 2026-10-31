// Purpose: Load the latest validated loop checkpoint without replaying argument drafts or effects.
// Pattern: Repository projection.
// Usage: Used during session recovery and explicit run continuation.
// Related: internal/storage/sqlite/runs_test.go, internal/agent/record/events.go.
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

func (s *Store) LatestRun(ctx context.Context, session string) (*runrecord.Checkpoint, error) {
	event, err := s.runState(ctx, session, "")
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if event.Checkpoint == nil {
		return nil, errors.New("run checkpoint is missing")
	}
	if err = event.Checkpoint.Check(); err != nil {
		return nil, err
	}
	return event.Checkpoint, nil
}
