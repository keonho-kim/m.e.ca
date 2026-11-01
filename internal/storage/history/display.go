// Purpose: Select and decode visible conversation fields without loading hidden native payloads.
// Pattern: Pure history projection independent of SQL and terminal state.
// Usage: Persistence adapters hydrate DisplayFields before passing visible bytes to Display.
// Related: internal/storage/history/record.go, internal/storage/sqlite/timeline.go.
package history

import (
	json "github.com/goccy/go-json"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/storage/content"
)

var latDisplayFields = content.Selection{"message": {
	"role": nil, "content": nil, "incomplete": nil, "reasoning": nil,
	"calls": {"*": {"id": nil, "name": nil}},
}}

// DisplayFields owns presentation policy; adapters retain ownership of payload I/O.
func DisplayFields(kind string, internal bool) (map[string]bool, content.Selection) {
	if kind == "message" {
		return map[string]bool{"role": true, "content": true, "incomplete": true}, nil
	}
	fields := map[string]bool{"kind": true, "text": true, "thinking": true, "call_id": true, "output": true, "operation": true, "change_preview": true, "checkpoint": true, "statistics": true, "internal": true}
	if internal {
		fields["lat"], fields["message"], fields["call"] = true, true, true
		return fields, latDisplayFields
	}
	return fields, nil
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
	if record.Event.Internal && record.Event.Call != nil {
		args, projectionErr := toolapi.DisplayArgs(record.Event.Call.Arguments)
		if projectionErr == nil {
			var err error
			record.Event.Call.Arguments, err = json.Marshal(args)
			if err != nil {
				return Record{}, err
			}
		}
		return record, nil
	}
	var call struct{ Call *struct{ ID, Name string } }
	if err := json.Unmarshal(envelope, &call); err != nil {
		return Record{}, err
	}
	if call.Call != nil {
		record.Event.Call = &toolapi.Call{ID: call.Call.ID, Name: call.Call.Name, Arguments: arguments}
	}
	return record, nil
}
