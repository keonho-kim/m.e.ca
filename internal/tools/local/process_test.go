//go:build linux || darwin

// Purpose: Verify failed shell commands keep structured output and an explicit failure state.
// Pattern: Process behavior tests.
// Usage: go test ./internal/tools/local -run TestCommandFailure.
// Related: internal/tools/local/process/run_unix.go, internal/agent/run/run.go.
package local

import (
	"context"
	"encoding/json"
	"github.com/keonho-kim/v-code/internal/policy"
	"testing"
)

func TestCommandFailurePreservesOutput(t *testing.T) {
	executor, _ := testExecutor(t)
	op, err := executor.Prepare(context.Background(), policy.Settings{Permission: policy.AllowAll, Scope: policy.System}, false, call("execute_command", `{"command":"printf output; exit 3"}`))
	if err != nil {
		t.Fatal(err)
	}
	result, err := op.Execute(context.Background(), true)
	if err != nil || !result.Failed {
		t.Fatalf("failure state: %+v %v", result, err)
	}
	var output struct {
		Stdout   string
		ExitCode int
	}
	if err = json.Unmarshal(result.Content, &output); err != nil {
		t.Fatal(err)
	}
	if output.Stdout != "output" || output.ExitCode != 3 {
		t.Fatalf("lost output: %+v", output)
	}
}
