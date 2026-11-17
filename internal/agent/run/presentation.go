// Purpose: Keep persisted and streamed generations and tool lifecycles at one display position.
// Pattern: Pure execution metadata projection.
// Usage: Journal writes and the runtime emitter share present before handing records to adapters.
// Related: internal/agent/run/journal.go, internal/agent/run/start.go, internal/agent/run/usage.go.
package run

import "github.com/keonho-kim/meca/internal/agent/record"

func (s *execution) present(event record.Event) record.Event {
	if event.Presentation != "" {
		return event
	}
	if event.Call != nil && event.Call.ExecutionID != "" {
		event.Presentation = event.Call.ExecutionID
		return event
	}
	switch event.Kind {
	case "generation_started", "stream", "thinking", "progress", "response":
		event.Presentation = s.presentation
	}
	return event
}
