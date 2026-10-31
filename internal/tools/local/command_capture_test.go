// Purpose: Keep completed command originals outside ordered parallel result queues until commit.
// Pattern: Command capture lifetime integration test.
// Usage: go test ./internal/tools/local -run TestCompletedCommandDefersCapture.
// Related: internal/tools/local/command_archive.go, internal/agent/run/execution_result.go.
package local

import (
	"context"
	"testing"

	"github.com/keonho-kim/meca/internal/policy"
)

func TestCompletedCommandDefersCaptureUntilCommit(t *testing.T) {
	e, _ := testExecutor(t)
	op, err := e.Prepare(context.Background(), policy.Settings{Scope: policy.System}, call("execute_command", `{"command":"head -c 4194304 /dev/zero"}`))
	if err != nil {
		t.Fatal(err)
	}
	result, err := op.Execute(context.Background(), true)
	if err != nil || result.Process == nil || result.Process.Status != "exited" {
		t.Fatalf("command failed: %+v %v", result.Process, err)
	}
	if len(result.Capture) != 0 || len(result.Content) > 1024*1024 {
		t.Fatalf("completed result retained its full original output: content=%d capture=%d", len(result.Content), len(result.Capture))
	}
	id := result.Process.CommandID
	if len(e.CommandActivity()) != 1 {
		t.Fatal("command was acknowledged before the runtime could commit it")
	}
	original, err := e.CommandCapture(id)
	if err != nil || len(original) < 4*1024*1024 {
		t.Fatal("deferred original missing", err)
	}
	if err = e.AcknowledgeCommand(id); err != nil {
		t.Fatal(err)
	}
	if len(e.CommandActivity()) != 0 {
		t.Fatal("committed command handle was retained")
	}
	if original, err = e.CommandCapture(id); err != nil || len(original) != 0 {
		t.Fatal("acknowledged capture was retained", err)
	}
}
