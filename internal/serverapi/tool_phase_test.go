// Purpose: Preserve command process liveness after public projections remove private operation details.
// Pattern: Cross-layer projection contract regression.
// Usage: go test ./internal/serverapi -run TestToolPhaseProjection.
// Related: internal/agent/record/tool_phase.go, internal/commands/server/projection.go.
package serverapi

import (
	"github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/commands/server"
	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
	"testing"
)

func TestToolPhaseProjectionPreservesActiveCommand(t *testing.T) {
	value := server.Event{Record: record.Event{Kind: "tool_succeeded", Call: &tool.Call{ID: "call", Name: "execute_command"}, Operation: &tool.Operation{Process: &tool.ProcessOutcome{Status: "running"}}}}
	projected := server.ProjectEvent(value)
	if projected.Record.Operation != nil {
		t.Fatal("private operation retained")
	}
	for range 2 {
		wire := event(projected)
		if wire.Tool.GetPhase() != v1.ToolPhase_TOOL_PHASE_RUNNING {
			t.Fatal("running process reported completion", wire.Tool)
		}
		projected = server.ProjectEvent(projected)
	}
}

func TestToolPhaseProjectionPreservesCommandCompletion(t *testing.T) {
	for _, tc := range []struct {
		status string
		phase  v1.ToolPhase
	}{
		{"tool_succeeded", v1.ToolPhase_TOOL_PHASE_SUCCEEDED}, {"tool_failed", v1.ToolPhase_TOOL_PHASE_FAILED},
	} {
		value := server.Event{Record: record.Event{Kind: "command_finished", Call: &tool.Call{ID: "call", Name: "execute_command"}, Operation: &tool.Operation{Status: tc.status, Process: &tool.ProcessOutcome{Status: "exited"}}}}
		for range 2 {
			value = server.ProjectEvent(value)
			if phase := event(value).Tool.GetPhase(); phase != tc.phase {
				t.Fatal(tc.status, phase)
			}
		}
	}
}
