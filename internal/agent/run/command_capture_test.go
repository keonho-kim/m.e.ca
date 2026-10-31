// Purpose: Preserve lazy command originals in the durable result before releasing the owned archive.
// Pattern: Runtime and local adapter integration test.
// Usage: go test ./internal/agent/run -run TestCommandCaptureCommits.
// Related: internal/agent/run/execution_result.go, internal/tools/local/command_archive.go.
package run

import (
	"context"
	"testing"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func TestCommandCaptureCommitsBeforeAcknowledgement(t *testing.T) {
	calls := []toolapi.Call{batchCall("capture", "execute_command", map[string]any{"command": "head -c 4194304 /dev/zero"})}
	r, tools, _, journal, _ := realBatchRunner(t, calls)
	result, err := r.Run(context.Background(), "Inspect the output", approve, emit)
	if err != nil || result.Status != runrecord.Answered {
		t.Fatalf("run failed: %+v %v", result, err)
	}
	found := false
	for _, event := range journal.events {
		if event.Kind == "tool_succeeded" && event.CallID == "capture" {
			found = true
			if len(event.Capture) < 4*1024*1024 {
				t.Fatal("durable event lost the deferred command original")
			}
		}
	}
	if !found || len(tools.CommandActivity()) != 0 {
		t.Fatal("command was not committed and acknowledged")
	}
}
