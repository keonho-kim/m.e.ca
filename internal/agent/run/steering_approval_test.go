// Purpose: Stop obsolete approval sequences before prompting for more candidates in the same wave.
// Pattern: Real prepared-file operations with scripted user steering.
// Usage: Runtime tests keep filesystem effects observable in temporary projects.
// Related: internal/agent/run/prepare_execution.go, internal/agent/run/batch_wave.go.
package run

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/policy"
	"github.com/keonho-kim/meca/internal/tools/local"
)

type approvalTools struct{ *local.Executor }
type approvalOperation struct{ tool.Execution }

func (e approvalTools) Prepare(ctx context.Context, settings policy.Settings, call tool.Call) (tool.Execution, error) {
	op, err := e.Executor.Prepare(ctx, settings, call)
	if err != nil {
		return nil, err
	}
	return approvalOperation{op}, nil
}

func (op approvalOperation) Approval() tool.Approval {
	value := op.Execution.Approval()
	value.Required = true
	return value
}

func (op approvalOperation) Release() { tool.Release(op.Execution) }

func TestSteeringStopsRemainingWaveApprovalsBeforeEffects(t *testing.T) {
	calls := []tool.Call{
		batchCall("a", "write_file", map[string]any{"path": "a", "content": "obsolete"}),
		batchCall("b", "write_file", map[string]any{"path": "b", "content": "obsolete"}),
	}
	r, tools, root, _, g := realBatchRunner(t, calls)
	r.Tools = approvalTools{tools}
	var pending []string
	r.PollSteering = func() []string { values := pending; pending = nil; return values }
	approvals := 0
	approve := func(context.Context, tool.Approval) (bool, error) {
		approvals++
		pending = []string{"Keep both files unchanged"}
		return true, nil
	}
	if _, err := r.Run(context.Background(), "Create both files", approve, emit); err != nil {
		t.Fatal(err)
	}
	if approvals != 1 || len(g.requests) != 2 {
		t.Fatal("obsolete wave continued asking for approval", approvals, len(g.requests))
	}
	for _, path := range []string{"a", "b"} {
		if _, err := os.Stat(filepath.Join(root, path)); !os.IsNotExist(err) {
			t.Fatal("an obsolete prepared file was written", path, err)
		}
	}
}
