// Purpose: Verify prepared resource ownership is released on completion, rejection and failed sibling preparation.
// Pattern: Runtime lifecycle contract tests.
// Usage: go test ./internal/agent/run -run TestPreparedResources.
// Related: internal/agent/tool/release.go, internal/agent/run/batch_prepare.go.
package run

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/policy"
)

type ownedOperation struct {
	once     sync.Once
	released *atomic.Int32
	approved bool
}

func (op *ownedOperation) Approval() toolapi.Approval {
	return toolapi.Approval{Name: "inspect_fixture", Required: op.approved}
}
func (op *ownedOperation) Execute(context.Context, bool) (toolapi.Result, error) {
	return toolapi.Result{Summary: "read", Content: []byte(`{"ok":true}`)}, nil
}
func (op *ownedOperation) Release() { op.once.Do(func() { op.released.Add(1) }) }

type ownedTools struct {
	executor
	released atomic.Int32
	failure  bool
}

func (e *ownedTools) Prepare(_ context.Context, _ policy.Settings, call toolapi.Call) (toolapi.Execution, error) {
	if e.failure && call.ID == "b" {
		return nil, errors.New("cannot freeze sibling")
	}
	return &ownedOperation{released: &e.released, approved: e.approval}, nil
}
func (e *ownedTools) ParallelBatch(_ context.Context, _ policy.Settings, calls []toolapi.Call, _ int) (toolapi.BatchPlan, error) {
	return toolapi.BatchPlan{Count: len(calls), PreparationConcurrency: 1}, nil
}
func TestPreparedResourcesReleaseOnEveryOutcome(t *testing.T) {
	for _, kind := range []string{"complete", "deny", "prepare failure"} {
		t.Run(kind, func(t *testing.T) {
			tools := &ownedTools{executor: executor{approval: kind == "deny"}, failure: kind == "prepare failure"}
			g := &parallelModel{calls: []toolapi.Call{batchCall("a", "inspect_fixture", map[string]any{"path": "a"}), batchCall("b", "inspect_fixture", map[string]any{"path": "b"})}}
			r := fixture(&generator{}, &executor{}, &journal{})
			r.Generator, r.Tools = g, tools
			_, err := r.Run(context.Background(), "inspect", func(context.Context, toolapi.Approval) (bool, error) { return false, nil }, emit)
			if kind == "deny" && err == nil {
				t.Fatal("denial was ignored")
			}
			expected := int32(2)
			if kind == "prepare failure" {
				expected = 1
			}
			if tools.released.Load() != expected {
				t.Fatalf("released=%d expected=%d error=%v", tools.released.Load(), expected, err)
			}
		})
	}
}
