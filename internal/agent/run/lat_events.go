// Purpose: Publish scoped LAT conversation events without duplicating their durable records.
// Pattern: Presentation event adapter.
// Usage: A read-only phase forwards its live native events to the parent event sink.
// Related: internal/agent/run/lat_fork.go, internal/tui/session/lat_output.go.
package run

import (
	"time"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

func (s *execution) forwardLATEvent(event runrecord.Event) error {
	switch event.Kind {
	case "usage":
		// The phase journal owns the only durable usage record.
		if event.Generation != nil {
			s.statistics.add(event.Usage, time.Duration(event.Generation.MeasuredMillis)*time.Millisecond)
		}
	case "usage_progress", "generation_started", "generation_finished", "thinking", "retry",
		"stream", "progress", "response", "tool_started", "tool_succeeded", "tool_failed",
		"tool_interrupted", "denied", "command_finished", "compaction_started", "compaction_finished":
	default:
		return nil
	}
	return s.emit(event)
}
