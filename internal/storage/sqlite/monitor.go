// Purpose: Restore ordered Monitor observations without replaying execution.
// Pattern: Repository projection.
// Usage: Session workers load these values before starting a new run.
// Related: internal/agent/run/events.go, internal/storage/sqlite/events.go, internal/tui/session/tasks.go.
package sqlite

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent/monitor"
	"github.com/keonho-kim/v-code/internal/agent/run"
)

func (s *Store) MonitorHistory(ctx context.Context, session string) (records []monitor.Record, err error) {
	rows, err := s.db.QueryContext(ctx, `SELECT body FROM tool_events WHERE session_id=? AND json_extract(body,'$.kind')='monitor' ORDER BY sequence DESC LIMIT 64`, session)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()
	for rows.Next() {
		var body []byte
		if err := rows.Scan(&body); err != nil {
			return nil, err
		}
		if len(body) > maxEventBytes {
			return nil, fmt.Errorf("Monitor record exceeds limit")
		}
		var event run.Event
		if err := json.Unmarshal(body, &event); err != nil {
			return nil, err
		}
		if event.Monitor == nil {
			return nil, fmt.Errorf("missing Monitor record")
		}
		if err := event.Monitor.Check(); err != nil {
			return nil, err
		}
		records = append(records, *event.Monitor)
	}
	for i, j := 0, len(records)-1; i < j; i, j = i+1, j-1 {
		records[i], records[j] = records[j], records[i]
	}
	return records, rows.Err()
}
