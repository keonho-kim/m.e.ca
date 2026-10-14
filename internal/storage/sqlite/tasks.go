// Purpose: Load the latest validated loop checkpoint without replaying argument drafts or effects.
// Pattern: Repository projection.
// Usage: Used during session recovery and explicit task continuation.
// Related: internal/storage/sqlite/tasks_test.go, internal/agent/run/events.go.
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent/run"
)

func (s *Store) LatestTask(ctx context.Context, session string) (*run.Checkpoint, error) {
	var body []byte
	err := s.db.QueryRowContext(ctx, "SELECT body FROM tool_events WHERE session_id=? AND json_extract(body,'$.kind')='task_checkpoint' ORDER BY sequence DESC LIMIT 1", session).Scan(&body)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(body) > 512*1024 {
		return nil, fmt.Errorf("task checkpoint exceeds limit")
	}
	var event run.Event
	if err = json.Unmarshal(body, &event); err != nil {
		return nil, err
	}
	if event.Checkpoint == nil {
		return nil, fmt.Errorf("task checkpoint is missing")
	}
	if err = event.Checkpoint.Check(); err != nil {
		return nil, err
	}
	return event.Checkpoint, nil
}
