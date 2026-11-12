// Purpose: Select and decode visible conversation fields without loading hidden native payloads.
// Pattern: Pure history projection independent of SQL and terminal state.
// Usage: Persistence adapters hydrate DisplayFields before passing visible bytes to Display.
// Related: internal/storage/history/record.go, internal/storage/sqlite/timeline.go.
package history

import (
	json "github.com/goccy/go-json"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

// DisplayFields owns presentation policy; adapters retain ownership of payload I/O.
func DisplayFields(kind string) map[string]bool {
	if kind == "message" {
		return map[string]bool{"id": true, "role": true, "content": true, "incomplete": true}
	}
	return map[string]bool{"agent": true, "final": true, "reference": true, "kind": true, "text": true, "team": true, "thinking": true, "call_id": true, "output": true, "operation": true, "change_preview": true, "checkpoint": true, "statistics": true, "skill": true, "occurred_at": true, "error_code": true, "run_id": true}
}

// Display restores visible values from hydrated body and a small original envelope.
// Arguments contains the already-selected public tool parameters, never source code.
func Display(sequence int64, kind string, body, envelope, arguments []byte) (Record, error) {
	record := Record{Sequence: sequence, Kind: kind}
	if kind == "message" {
		if err := json.Unmarshal(body, &record.Message); err != nil {
			return Record{}, err
		}
		return record, record.Message.Check()
	}
	if err := json.Unmarshal(body, &record.Event); err != nil {
		return Record{}, err
	}
	var call struct {
		Call *struct {
			ID, Name    string
			ExecutionID string `json:"execution_id"`
			BatchID     string `json:"batch_id"`
		}
	}
	if err := json.Unmarshal(envelope, &call); err != nil {
		return Record{}, err
	}
	if call.Call != nil {
		record.Event.Call = &toolapi.Call{ExecutionID: call.Call.ExecutionID, BatchID: call.Call.BatchID, ID: call.Call.ID, Name: call.Call.Name, Arguments: arguments}
	}
	return record, nil
}
