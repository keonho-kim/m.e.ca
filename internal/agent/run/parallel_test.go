// Purpose: Verify independent native calls overlap, remain bounded and preserve ordered results.
// Pattern: Real filesystem execution with channel-controlled concurrency.
// Usage: go test -race ./internal/agent/run -run TestParallel.
// Related: internal/agent/run/batch.go, internal/agent/run/batch_wave.go, internal/tools/local/parallel.go.
package run

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/policy"
	"github.com/keonho-kim/meca/internal/tools/local"
)

type parallelModel struct {
	calls    []toolapi.Call
	requests []modelapi.Request
}

func (g *parallelModel) Generate(_ context.Context, r modelapi.Request, emit func(modelapi.Event) error) error {
	g.requests = append(g.requests, r)
	if len(g.requests) == 1 {
		return emit(modelapi.Event{Calls: g.calls, Done: true})
	}
	return emit(modelapi.Event{Delta: "Completed requested operations", Done: true})
}
func batchCall(id, name string, args map[string]any) toolapi.Call {
	body, err := json.Marshal(args)
	if err != nil {
		panic(err)
	}
	return toolapi.Call{ID: id, Name: name, Arguments: body}
}

type heldTools struct {
	*local.Executor
	entered         chan string
	release         chan struct{}
	active, maximum atomic.Int32
}
type heldOperation struct {
	toolapi.Execution
	owner *heldTools
	id    string
}

func (e *heldTools) Prepare(ctx context.Context, settings policy.Settings, call toolapi.Call) (toolapi.Execution, error) {
	op, err := e.Executor.Prepare(ctx, settings, call)
	if err != nil {
		return nil, err
	}
	return heldOperation{op, e, call.ID}, nil
}
func (op heldOperation) Execute(ctx context.Context, approved bool) (toolapi.Result, error) {
	current := op.owner.active.Add(1)
	defer op.owner.active.Add(-1)
	for old := op.owner.maximum.Load(); current > old; old = op.owner.maximum.Load() {
		if op.owner.maximum.CompareAndSwap(old, current) {
			break
		}
	}
	op.owner.entered <- op.id
	select {
	case <-ctx.Done():
		return toolapi.Result{}, ctx.Err()
	case <-op.owner.release:
		return op.Execution.Execute(ctx, approved)
	}
}
func realBatchRunner(t *testing.T, calls []toolapi.Call) (Runner, *local.Executor, string, *journal, *parallelModel) {
	t.Helper()
	root, home := t.TempDir(), t.TempDir()
	tools, err := local.NewExecutor(root, home, filepath.Join(home, ".meca"), func(s string) string { return s })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := tools.Close(); err != nil {
			t.Error(err)
		}
	})
	j, g := &journal{}, &parallelModel{calls: calls}
	r := fixture(&generator{}, &executor{}, j)
	r.Tools, r.Generator = tools, g
	r.Settings = policy.Settings{Scope: policy.System}
	return r, tools, root, j, g
}
func TestParallelBoundedExecutionAndOrderedResults(t *testing.T) {
	var calls []toolapi.Call
	for i := 0; i < 18; i++ {
		calls = append(calls, batchCall(fmt.Sprint(i), "create_directory", map[string]any{"paths": []string{fmt.Sprintf("dir%d", i)}}))
	}
	r, tools, root, j, model := realBatchRunner(t, calls)
	held := &heldTools{Executor: tools, entered: make(chan string, 18), release: make(chan struct{})}
	r.Tools = held
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	done := make(chan struct{})
	var result Result
	var runErr error
	go func() { defer close(done); result, runErr = r.Run(ctx, "create eighteen directories", approve, emit) }()
	t.Cleanup(func() { cancel(); <-done })
	for i := 0; i < 16; i++ {
		select {
		case <-held.entered:
		case <-ctx.Done():
			t.Fatal("sixteen operations did not overlap")
		}
	}
	close(held.release)
	<-done
	if runErr != nil || result.Status != runrecord.Answered || held.maximum.Load() != 16 {
		t.Fatalf("%+v active max=%d err=%v", result, held.maximum.Load(), runErr)
	}
	index := 0
	for _, m := range j.messages {
		if m.Role == "tool" {
			if m.ToolCallID != fmt.Sprint(index) {
				t.Fatalf("result order=%s want=%d", m.ToolCallID, index)
			}
			index++
		}
	}
	if len(model.requests) != 2 {
		t.Fatalf("unexpected main model calls: %d", len(model.requests))
	}
	t.Logf("operations=18, peak concurrency=%d, main model calls=%d", held.maximum.Load(), len(model.requests))
	if index != 18 {
		t.Fatalf("paired results=%d", index)
	}
	for i := 0; i < 18; i++ {
		if _, err := os.Stat(filepath.Join(root, fmt.Sprintf("dir%d", i))); err != nil {
			t.Fatal(err)
		}
	}
}
func TestParallelSerializesDependentTargets(t *testing.T) {
	calls := []toolapi.Call{
		batchCall("parent", "create_directory", map[string]any{"paths": []string{"dir"}}),
		batchCall("first", "write_file", map[string]any{"path": "dir/a", "content": "one"}),
		batchCall("second", "write_file", map[string]any{"path": "dir/a", "content": "two"}),
	}
	r, _, root, _, _ := realBatchRunner(t, calls)
	result, err := r.Run(context.Background(), "create then update", approve, emit)
	if err != nil || result.Status != runrecord.Answered {
		t.Fatalf("%+v %v", result, err)
	}
	body, err := os.ReadFile(filepath.Join(root, "dir/a"))
	if err != nil || strings.TrimSpace(string(body)) != "two" {
		t.Fatalf("%q %v", body, err)
	}
}
func TestParallelCancellationJoinsWorkersAndPairsResults(t *testing.T) {
	calls := []toolapi.Call{batchCall("a", "create_directory", map[string]any{"paths": []string{"a"}}), batchCall("b", "create_directory", map[string]any{"paths": []string{"b"}})}
	r, tools, _, j, _ := realBatchRunner(t, calls)
	held := &heldTools{Executor: tools, entered: make(chan string, 2), release: make(chan struct{})}
	r.Tools = held
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	done := make(chan struct{})
	var result Result
	var runErr error
	go func() {
		defer close(done)
		result, runErr = r.Run(ctx, "create two directories", approve, func(runrecord.Event) error { return ctx.Err() })
	}()
	t.Cleanup(func() { cancel(); <-done })
	for i := 0; i < 2; i++ {
		select {
		case <-held.entered:
		case <-ctx.Done():
			t.Fatal("workers did not start")
		}
	}
	cancel()
	<-done
	if runErr == nil || result.Status != runrecord.Interrupted || held.active.Load() != 0 {
		t.Fatalf("%+v active=%d %v", result, held.active.Load(), runErr)
	}
	count := 0
	for _, m := range j.messages {
		if m.Role == "tool" {
			count++
		}
	}
	if count != 2 {
		t.Fatalf("cancelled calls lost their results: %d", count)
	}
}

func TestParallelApprovalDenialPreventsWaveEffects(t *testing.T) {
	root := t.TempDir()
	calls := []toolapi.Call{batchCall("a", "create_directory", map[string]any{"paths": []string{filepath.Join(root, "a")}}), batchCall("b", "create_directory", map[string]any{"paths": []string{filepath.Join(root, "b")}})}
	r, _, _, j, _ := realBatchRunner(t, calls)

	decisions := 0
	result, err := r.Run(context.Background(), "create both", func(context.Context, toolapi.Approval) (bool, error) { decisions++; return decisions == 1, nil }, emit)
	if err == nil || result.Status != runrecord.Stopped || decisions != 2 {
		t.Fatalf("%+v decisions=%d %v", result, decisions, err)
	}
	for _, name := range []string{"a", "b"} {
		if _, err := os.Stat(filepath.Join(root, name)); !os.IsNotExist(err) {
			t.Fatal("wave executed before all approvals")
		}
	}
	count := 0
	for _, m := range j.messages {
		if m.Role == "tool" {
			count++
		}
	}
	if count != 2 {
		t.Fatalf("denied batch lost paired results: %d", count)
	}
}

func TestParallelRechecksAliasesAfterApproval(t *testing.T) {
	root := t.TempDir()
	calls := []toolapi.Call{batchCall("a", "write_file", map[string]any{"path": filepath.Join(root, "a"), "content": "changed"}), batchCall("b", "write_file", map[string]any{"path": filepath.Join(root, "b"), "content": "changed"})}
	r, _, _, _, _ := realBatchRunner(t, calls)
	for _, name := range []string{"a", "b"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("original"), 0600); err != nil {
			t.Fatal(err)
		}
	}

	decisions := 0
	result, err := r.Run(context.Background(), "update both", func(context.Context, toolapi.Approval) (bool, error) {
		decisions++
		if decisions == 2 {
			if err := os.Remove(filepath.Join(root, "b")); err != nil {
				return false, err
			}
			if err := os.Link(filepath.Join(root, "a"), filepath.Join(root, "b")); err != nil {
				return false, err
			}
		}
		return true, nil
	}, emit)
	if err != nil || !result.HadToolFailure {
		t.Fatalf("%+v %v", result, err)
	}
	body, err := os.ReadFile(filepath.Join(root, "a"))
	if err != nil || string(body) != "original" {
		t.Fatalf("alias conflict executed: %q %v", body, err)
	}
}

func TestParallelWritesKeepIndividualApprovals(t *testing.T) {
	root := t.TempDir()
	calls := []toolapi.Call{batchCall("a", "write_file", map[string]any{"path": filepath.Join(root, "a"), "content": "first"}), batchCall("b", "write_file", map[string]any{"path": filepath.Join(root, "b"), "content": "second"})}
	r, tools, _, _, _ := realBatchRunner(t, calls)

	held := &heldTools{Executor: tools, entered: make(chan string, 2), release: make(chan struct{})}
	r.Tools = held
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	done := make(chan struct{})
	var result Result
	var runErr error
	decisions := 0
	go func() {
		defer close(done)
		result, runErr = r.Run(ctx, "write both files", func(context.Context, toolapi.Approval) (bool, error) { decisions++; return true, nil }, emit)
	}()
	t.Cleanup(func() { cancel(); <-done })
	for i := 0; i < 2; i++ {
		select {
		case <-held.entered:
		case <-ctx.Done():
			t.Fatal("writes did not overlap")
		}
	}
	close(held.release)
	<-done
	if runErr != nil || result.Status != runrecord.Answered || decisions != 2 || held.maximum.Load() != 2 {
		t.Fatalf("%+v approvals=%d concurrent=%d %v", result, decisions, held.maximum.Load(), runErr)
	}
	for _, name := range []string{"a", "b"} {
		if _, err := os.Stat(filepath.Join(root, name)); err != nil {
			t.Fatal(err)
		}
	}
}
