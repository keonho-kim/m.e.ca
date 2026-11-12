// Purpose: Distinguish a returned command observation from its still-running process.
// Pattern: Pure execution-fact regression.
// Usage: go test ./internal/agent/record -run TestToolPhase.
// Related: internal/agent/record/tool_phase.go, internal/serverapi/projection.go.
package record

import (
	"github.com/keonho-kim/meca/internal/agent/tool"
	"testing"
)

func TestToolPhase(t *testing.T) {
	for _, state := range []string{"waiting", "running", "finalizing"} {
		event := Event{Kind: "tool_succeeded", Operation: &tool.Operation{Process: &tool.ProcessOutcome{Status: state}}}
		if event.ToolPhase() != ToolRunning {
			t.Fatal(state, event.ToolPhase())
		}
	}
	if (Event{Kind: "tool_failed"}).ToolPhase() != ToolFailed {
		t.Fatal("failed call was not terminal")
	}
	if (Event{Kind: "tool_succeeded"}).ToolPhase() != ToolSucceeded {
		t.Fatal("ordinary success was not terminal")
	}
}

func TestToolPhaseCommandCompletion(t *testing.T) {
	for _, tc := range []struct {
		status string
		phase  ToolPhase
	}{
		{"tool_succeeded", ToolSucceeded}, {"tool_failed", ToolFailed}, {"", ToolUnknown},
	} {
		event := Event{Kind: "command_finished", Operation: &tool.Operation{Status: tc.status, Process: &tool.ProcessOutcome{Status: "exited"}}}
		if phase := event.ToolPhase(); phase != tc.phase {
			t.Fatal(tc.status, phase, tc.phase)
		}
	}
	if phase := (Event{Kind: "command_finished"}).ToolPhase(); phase != ToolUnknown {
		t.Fatal(phase)
	}
}
