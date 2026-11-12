// Purpose: Project finished checkpoints into measurement-only public history entries.
// Pattern: Pure adapter with an explicit terminal-status mapping.
// Usage: historyItem returns this payload before considering any narrative fields.
// Related: internal/serverapi/history_content.go, api/meca/v1/events.proto, internal/serverapi/state_values.go.
package serverapi

import (
	"github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/work"
	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
)

func historyCompletion(event record.Event) *v1.RunCompletion {
	checkpoint := event.Checkpoint
	if event.Kind != "run_checkpoint" || checkpoint == nil || checkpoint.Phase != "finished" || checkpoint.End == nil {
		return nil
	}
	value := &v1.RunCompletion{}
	switch checkpoint.End.Status {
	case work.Completed:
		value.Status = v1.RunCompletion_COMPLETED
	case work.Failed:
		value.Status = v1.RunCompletion_FAILED
	case work.Interrupted:
		value.Status = v1.RunCompletion_INTERRUPTED
	case work.Denied:
		value.Status = v1.RunCompletion_DENIED
	case work.Exhausted:
		value.Status = v1.RunCompletion_EXHAUSTED
	}
	if event.Statistics != nil {
		value.Statistics = wireStatistics(*event.Statistics)
	}
	return value
}
