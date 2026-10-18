// Purpose: Keep repeated command output variations from resetting request progress.
// Pattern: Deterministic progress regression tests.
// Usage: go test ./internal/agent/run -run TestCommandProgress.
// Related: internal/agent/run/semantic_progress.go, internal/agent/progress/request.go.
package run

import (
	"encoding/json"
	"testing"

	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/policy"
)

func TestCommandProgressRequiresNewInvocationAndObservation(t *testing.T) {
	s := execution{}
	observe := func(command, output string) bool {
		args, _ := json.Marshal(map[string]string{"command": command})
		body, _ := json.Marshal(map[string]string{"Stdout": output})
		return s.toolAdvanced(toolapi.Call{Name: "execute_command", Arguments: args}, toolapi.Result{Content: body, Process: &toolapi.ProcessOutcome{Status: "exited"}}, policy.Command, "")
	}
	if !observe("pytest | tail", "ERROR: syntax in 0.24s") {
		t.Fatal("first observation lost")
	}
	if observe("pytest | tail", "ERROR: syntax in 0.19s") {
		t.Fatal("same command and source fabricated progress from changed output")
	}
	if observe("another spelling", "ERROR: syntax in 0.24s") {
		t.Fatal("new command spelling fabricated progress from the same output")
	}
	s.checkpoint.Work.CurrentFiles = []toolapi.FileFact{{Path: "/app.py", Version: "corrected", Observation: "present"}}
	if !observe("pytest | tail", "ERROR: syntax in 0.19s") {
		t.Fatal("changed source did not permit a fresh observation")
	}
}
