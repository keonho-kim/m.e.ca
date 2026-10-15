// Purpose: Project persisted history into ordered compaction evidence without database access.
// Pattern: Pure projection.
// Usage: SQLite passes one decoded page; compaction consumes the resulting records.
// Related: internal/storage/history/record.go, internal/agent/compact/types.go.
package history

import (
	"encoding/json"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent/compact"
	"sort"
)

func Compaction(page []Record) ([]compact.Record, error) {
	var records []compact.Record
	runs := map[string]compact.Record{}
	for _, record := range page {
		sequence := record.Sequence
		if record.Kind == "message" {
			m := record.Message
			if m.Content != "" || len(m.Calls) > 0 {
				body, err := json.Marshal(m)
				if err != nil {
					return nil, err
				}
				records = append(records, compact.Record{Sequence: sequence, Role: m.Role, Text: string(body)})
			}
			continue
		}
		event := record.Event
		if event.Kind == "conclave" && event.Conclave != nil {
			if err := event.Conclave.Check(); err != nil {
				return nil, err
			}
			records = append(records, compact.Record{Sequence: sequence, Role: "conclave", Review: event.Conclave})
		}
		if event.Monitor != nil {
			if err := event.Monitor.Check(); err != nil {
				return nil, err
			}
			encoded, err := json.Marshal(event.Monitor)
			if err != nil {
				return nil, err
			}
			records = append(records, compact.Record{Sequence: sequence, Role: "monitor", Text: string(encoded)})
		}
		if event.Grounding != nil {
			if err := event.Grounding.Check(); err != nil {
				return nil, err
			}
			records = append(records, compact.Record{Sequence: sequence, Role: "self_grounding", Grounding: event.Grounding})
		}
		if cp := event.Checkpoint; cp != nil {
			fact := compact.RunFact{Checks: cp.Checks, ChangeRevision: cp.ChangeRevision, ID: cp.ID, Objective: cp.Objective, Status: string(cp.Status), Phase: cp.Phase, Evidence: cp.Evidence, Observation: cp.Observation, Step: cp.Step, Source: sequence}
			runs[cp.ID] = compact.Record{Sequence: sequence, Role: "run", Text: cp.Objective, Run: &fact}
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

	for _, record := range runs {
		records = append(records, record)
	}
	sort.Slice(records, func(i, j int) bool { return records[i].Sequence < records[j].Sequence })
	return records, nil
}
