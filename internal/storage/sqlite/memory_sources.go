// Purpose: Project chronological conversation and runtime evidence into bounded compaction source pages.
// Pattern: Repository projection.
// Usage: Called by the compaction controller; raw code, drafts and original output remain archived.
// Related: internal/storage/sqlite/memory.go, internal/agent/compact/types.go, internal/agent/run/types.go.
package sqlite

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent/compact"
	"github.com/keonho-kim/v-code/internal/agent/run"
	"sort"
)

func (s *Store) MemoryRecords(ctx context.Context, session string, after, through int64) (records []compact.Record, cursor int64, err error) {
	rows, err := s.db.QueryContext(ctx, `SELECT t.sequence,t.kind,COALESCE(json_extract(m.body,'$.role'),''),COALESCE(json_extract(m.body,'$.content'),''),e.body FROM timeline t LEFT JOIN messages m ON t.kind='message' AND m.sequence=t.source_sequence LEFT JOIN tool_events e ON t.kind='event' AND e.sequence=t.source_sequence WHERE t.session_id=? AND t.sequence>? AND t.sequence<=? ORDER BY t.sequence LIMIT 64`, session, after, through)
	if err != nil {
		return nil, after, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()
	tasks := map[string]compact.Record{}
	bytes := 0
	cursor = after
	for rows.Next() {
		var sequence int64
		var kind, role, text string
		var body []byte
		if err = rows.Scan(&sequence, &kind, &role, &text, &body); err != nil {
			return nil, cursor, err
		}
		bytes += len(body) + len(text)
		if bytes > 8*1024*1024 && cursor > after {
			break
		}
		cursor = sequence
		if kind == "message" {
			if (role == "user" || role == "assistant") && text != "" {
				records = append(records, compact.Record{Sequence: sequence, Role: role, Text: text})
			}
			continue
		}
		var event run.Event
		if err = json.Unmarshal(body, &event); err != nil {
			return nil, cursor, err
		}
		if cp := event.Checkpoint; cp != nil {
			fact := compact.TaskFact{ID: cp.ID, Objective: cp.Objective, Status: string(cp.Status), Phase: cp.Phase, Evidence: cp.Evidence, Observation: cp.Observation, Step: cp.Step, Source: sequence}
			tasks[cp.ID] = compact.Record{Sequence: sequence, Role: "task", Text: cp.Objective, Task: &fact}
		}
		if event.Operation == nil && event.Text != "" && event.Kind != "thinking" && event.Kind != "compaction" {
			records = append(records, compact.Record{Sequence: sequence, Role: "runtime_event", Text: fmt.Sprintf("%s; operation=%s; %s", event.Kind, event.CallID, event.Text)})
		}
		if event.Operation != nil {
			op := *event.Operation
			op.Source = sequence
			records = append(records, compact.Record{Sequence: sequence, Role: "operation", Text: fmt.Sprintf("%s: %s; %s", op.Name, op.Status, op.Result), Operation: &op})
		}
	}
	for _, record := range tasks {
		records = append(records, record)
	}
	sort.Slice(records, func(i, j int) bool { return records[i].Sequence < records[j].Sequence })
	return records, cursor, rows.Err()
}
