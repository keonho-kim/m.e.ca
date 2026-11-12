// Purpose: Derive one tool lifecycle phase from the recorded invocation and process facts.
// Pattern: Pure execution-state projection.
// Usage: Server transports and terminal summaries share this decision.
// Related: internal/agent/record/events.go, internal/agent/tool/commands.go.
package record

import "github.com/keonho-kim/meca/internal/agent/tool"

type ToolPhase string

const (
	ToolUnknown     ToolPhase = "unknown"
	ToolPending     ToolPhase = "pending"
	ToolRunning     ToolPhase = "running"
	ToolSucceeded   ToolPhase = "succeeded"
	ToolFailed      ToolPhase = "failed"
	ToolInterrupted ToolPhase = "interrupted"
	ToolSkipped     ToolPhase = "skipped"
)

func (e Event) ToolPhase() ToolPhase {
	kind := e.Kind
	if kind == "command_finished" {
		if e.Operation == nil {
			return ToolUnknown
		}
		kind = e.Operation.Status
	}
	switch kind {
	case "tool_failed", "denied":
		return ToolFailed
	case "tool_interrupted":
		return ToolInterrupted
	case "tool_skipped":
		return ToolSkipped
	case "tool_started":
		return ToolRunning
	case "tool_proposed", "approved", "tool_waiting":
		return ToolPending
	case "tool_succeeded", "skill_loaded":
		if e.Operation != nil && e.Operation.Process != nil && tool.CommandActive(e.Operation.Process.Status) {
			return ToolRunning
		}
		return ToolSucceeded
	default:
		return ToolUnknown
	}
}
