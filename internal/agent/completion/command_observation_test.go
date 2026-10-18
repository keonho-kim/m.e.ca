// Purpose: Preserve failed and nonmatching command outcomes as historical observations.
// Pattern: Pure state contract test.
// Usage: go test ./internal/agent/completion -run TestCommandObservation.
// Related: internal/agent/completion/state.go, internal/agent/completion/facts.go.
package completion

import "testing"

func TestCommandObservationPreservesNonzeroExit(t *testing.T) {
	one := 1
	var state State
	state.Observe(Receipt{ID: "search", Tool: "execute_command", Kind: "command", Command: "grep TODO app.py", ProcessStatus: "exited", ExitCode: &one, Success: false, Output: "no matches"}, nil, true)
	facts := state.Facts()
	if len(facts.Commands) != 1 || facts.Commands[0].ExitCode == nil || *facts.Commands[0].ExitCode != 1 || state.Receipts[0].Success || facts.Commands[0].Output != "no matches" {
		t.Fatalf("command facts changed: %+v", facts)
	}
}
