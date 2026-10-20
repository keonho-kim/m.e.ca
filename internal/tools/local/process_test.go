//go:build linux || darwin

// Purpose: Verify shell failures and caller deadlines preserve output and explicit failure state.
// Pattern: Process behavior tests.
// Usage: go test ./internal/tools/local -run TestCommand.
// Related: internal/tools/local/process/run_unix.go, internal/agent/run/run.go.
package local

import (
	"context"
	"encoding/json"
	"errors"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/policy"
	"testing"
	"time"
)

func TestCommandFailurePreservesOutput(t *testing.T) {
	executor, _ := testExecutor(t)
	op, err := executor.Prepare(context.Background(), policy.Settings{Scope: policy.System}, call("execute_command", `{"command":"printf output; exit 3"}`))
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

func TestCommandTaskDeadlinePreservesPartialOutput(t *testing.T) {
	executor, _ := testExecutor(t)
	op, err := executor.Prepare(context.Background(), policy.Settings{Scope: policy.System}, call("execute_command", `{"command":"printf partial; sleep 30"}`))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	result, err := op.Execute(ctx, true)
	if !errors.Is(err, context.DeadlineExceeded) || !result.Failed || result.Failure == nil || result.Failure.Code != toolapi.Timeout {
		t.Fatalf("timeout=%+v %v", result, err)
	}
	var output struct{ Stdout, Status string }
	if err := json.Unmarshal(result.Content, &output); err != nil {
		t.Fatal(err)
	}
	if output.Stdout != "partial" || output.Status != "timeout" {
		t.Fatalf("partial output lost: %+v", output)
	}
}
