// Purpose: Translate execution phases into the versioned native tool contract.
// Pattern: Transport projection adapter.
// Usage: Event projection uses this for live and restored tool observations.
// Related: internal/agent/record/tool_phase.go, api/meca/v1/events.proto.
package serverapi

import (
	"github.com/keonho-kim/meca/internal/agent/record"
	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
)

func wireToolPhase(phase record.ToolPhase) v1.ToolPhase {
	switch phase {
	case record.ToolPending:
		return v1.ToolPhase_TOOL_PHASE_PENDING
	case record.ToolRunning:
		return v1.ToolPhase_TOOL_PHASE_RUNNING
	case record.ToolSucceeded:
		return v1.ToolPhase_TOOL_PHASE_SUCCEEDED
	case record.ToolFailed:
		return v1.ToolPhase_TOOL_PHASE_FAILED
	case record.ToolInterrupted:
		return v1.ToolPhase_TOOL_PHASE_INTERRUPTED
	case record.ToolSkipped:
		return v1.ToolPhase_TOOL_PHASE_SKIPPED
	default:
		return v1.ToolPhase_TOOL_PHASE_UNSPECIFIED
	}
}
