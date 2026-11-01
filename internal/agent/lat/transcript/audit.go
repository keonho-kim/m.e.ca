// Purpose: Keep durable phase evidence while omitting redundant execution bookkeeping.
// Pattern: Pure audit projection.
// Usage: Journal.Commit projects internal events before its parent durability boundary.
// Related: internal/agent/lat/transcript/journal.go, internal/agent/record/checkpoint.go.
package transcript

import (
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func auditEvent(event runrecord.Event) (runrecord.Event, bool) {
	switch event.Kind {
	case "tool_proposed", "tool_started", "compaction":
		// The native call records the candidate; actual outcome events preserve
		// execution evidence. Private summaries have their own lat_compaction audit.
		return runrecord.Event{}, false
	case "run_checkpoint":
		if event.Checkpoint.Phase != "finished" {
			return runrecord.Event{}, false
		}
		checkpoint := *event.Checkpoint
		// Side sessions are ephemeral, never independently resumed. Their native
		// request and outcome messages already own these potentially large bodies.
		checkpoint.Objective, checkpoint.Work = "", toolapi.Facts{}
		event.Checkpoint, event.Statistics = &checkpoint, nil
		event.Kind = "lat_checkpoint"
	}
	return event, true
}
