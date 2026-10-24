// Purpose: Load the latest validated loop checkpoint without replaying argument drafts or effects.
// Pattern: Repository projection.
// Usage: Used during session recovery and explicit run continuation.
// Related: internal/storage/sqlite/runs_test.go, internal/agent/run/events.go.
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/keonho-kim/meca/internal/agent/run"
)

func (s *Store) LatestRun(ctx context.Context, session string) (*run.Checkpoint, error) {
	var record storedRecord
	err := s.db.QueryRowContext(ctx, "SELECT body,refs,raw_bytes FROM tool_events WHERE session_id=? AND kind='run_checkpoint' ORDER BY sequence DESC LIMIT 1", session).Scan(&record.body, &record.refs, &record.size)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if record.size > maxEventBytes {
		return nil, fmt.Errorf("run checkpoint exceeds limit")
	}
	body, err := s.unpackRecord(ctx, record, nil)
	if err != nil {
		return nil, err
	}
	var event run.Event
	if err = json.Unmarshal(body, &event); err != nil {
		return nil, err
	}
	if event.Checkpoint == nil {
		return nil, fmt.Errorf("run checkpoint is missing")
	}
	if err = event.Checkpoint.Check(); err != nil {
		return nil, err
	}
	return event.Checkpoint, nil
}
