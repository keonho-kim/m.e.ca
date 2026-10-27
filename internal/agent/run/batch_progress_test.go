// Purpose: Verify ordered ready results become visible before unrelated parallel work finishes.
// Pattern: Channel-controlled fork/join regression test.
// Usage: go test -race ./internal/agent/run -run TestParallelPublishesReadyResults.
// Related: internal/agent/run/batch_wave.go, internal/agent/run/parallel_test.go.
package run

import (
	"context"
	"testing"
	"time"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/policy"
	"github.com/keonho-kim/meca/internal/tools/local"
)

type progressTools struct {
	*local.Executor
	release chan struct{}
}

type progressOperation struct {
	toolapi.Execution
	release <-chan struct{}
}

func (e *progressTools) Prepare(ctx context.Context, settings policy.Settings, call toolapi.Call) (toolapi.Execution, error) {
	op, err := e.Executor.Prepare(ctx, settings, call)
	if err == nil && call.ID == "checks" {
		return progressOperation{op, e.release}, nil
	}
	return op, err
}

func (op progressOperation) Execute(ctx context.Context, approved bool) (toolapi.Result, error) {
	select {
	case <-op.release:
		return op.Execution.Execute(ctx, approved)
	case <-ctx.Done():
		return toolapi.Result{}, ctx.Err()
	}
}

func TestParallelPublishesReadyResults(t *testing.T) {
	calls := []toolapi.Call{
		batchCall("read", "execute_command", map[string]any{"command": "printf document-read"}),
		batchCall("checks", "execute_command", map[string]any{"command": "printf checks-finished"}),
	}
	r, tools, _, journal, _ := realBatchRunner(t, calls)
	held := &progressTools{Executor: tools, release: make(chan struct{})}
	r.Tools = held
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	visible := false
	result, err := r.Run(ctx, "read documents and run checks independently", approve, func(event runrecord.Event) error {
		if event.Kind == "tool_succeeded" && event.CallID == "read" && !visible {
			visible = true
			close(held.release)
		}
		return nil
	})
	if err != nil || result.Status != runrecord.Answered || !visible || ctx.Err() != nil {
		t.Fatalf("first result waited for its blocked sibling: result=%+v visible=%v error=%v", result, visible, err)
	}
	var paired []string
	for _, message := range journal.messages {
		if message.Role == "tool" {
			paired = append(paired, message.ToolCallID)
		}
	}
	if len(paired) != 2 || paired[0] != "read" || paired[1] != "checks" {
		t.Fatalf("native result order changed: %v", paired)
	}
}
