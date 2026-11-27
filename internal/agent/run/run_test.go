// Purpose: Verify native tool execution, model-call budgets and final reporting.
// Pattern: Task behavior tests with scripted model replies.
// Usage: go test ./internal/agent/run.
// Related: internal/agent/run/run.go, internal/agent/run/codemode.go.
package run

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/policy"
)

func TestDeniedToolStopsAndPersistsStructuredResult(t *testing.T) {
	g, e, j := &generator{replies: []string{nativeCall("write_file", map[string]any{"path": "file.txt", "content": "new"})}}, &executor{approval: true}, &journal{}
	r := fixture(g, e, j)
	result, err := r.Run(context.Background(), "change", func(context.Context, toolapi.Approval) (bool, error) { return false, nil }, emit)
	if !errors.Is(err, errApprovalDenied) || e.executed != 0 || e.prepared != 1 || result.Status != runrecord.Stopped || !result.HadToolFailure {
		t.Fatalf("denial: %+v %v", result, err)
	}
	if len(g.requests) != 1 || len(j.messages) != 3 || j.messages[2].Role != "tool" {
		t.Fatalf("audit: %+v", j.messages)
	}
	var args map[string]string
	if err := json.Unmarshal(e.calls[0].Arguments, &args); err != nil {
		t.Fatal(err)
	}
	if args["path"] != "file.txt" || args["content"] != "new" {
		t.Fatal(args)
	}
}
func TestIncompleteArgumentsNeverPrepareOrExecute(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	g := &generator{replies: []string{nativeCall("write_file", map[string]any{"path": "file.txt", "content": "unfinished"})}, cancelAt: 1, cancel: cancel}
	e, j := &executor{}, &journal{}
	r := fixture(g, e, j)
	result, err := r.Run(ctx, "change", approve, emit)
	if !errors.Is(err, context.Canceled) || e.prepared != 0 || e.executed != 0 || result.Status != runrecord.Interrupted {
		t.Fatalf("partial argument: %+v %v", result, err)
	}
	var last *runrecord.Checkpoint
	for _, event := range j.events {
		if event.Checkpoint != nil {
			last = event.Checkpoint
		}
	}
	if last == nil || last.Status != runrecord.Interrupted || last.Tool != "" {
		t.Fatalf("checkpoint: %+v", last)
	}
}
func TestChangesAllowFinalReportWithoutCheck(t *testing.T) {
	g := &generator{replies: []string{nativeCall("write_file", map[string]any{"path": "file.txt", "content": "new"}), "File changed; tests were not run"}}
	e, j := &executor{}, &journal{}
	r := fixture(g, e, j)
	result, err := r.Run(context.Background(), "change", approve, emit)
	if err != nil || result.Status != runrecord.Answered {
		t.Fatalf("unchecked completion: %+v %v", result, err)
	}
}
func TestVerifiedCommandAllowsCompletionAndFullContexts(t *testing.T) {
	g := &generator{replies: []string{nativeCall("write_file", map[string]any{"path": "file.txt", "content": "new"}), nativeCall("execute_command", map[string]any{"command": "go test ./..."}), "The check passed"}}
	e, j := &executor{}, &journal{}
	r := fixture(g, e, j)
	r.Settings.Scope = policy.System
	result, err := r.Run(context.Background(), "change", approve, emit)
	if err != nil || result.Status != runrecord.Answered {
		t.Fatalf("completion: %+v %v", result, err)
	}
	for _, request := range g.requests {
		if len(request.Messages) < 2 || !strings.Contains(request.Messages[0].Content, r.ProjectInstructions) {
			t.Fatal("rules or full context lost")
		}
	}
}
func TestLongRequestRetainsActionsAndHistory(t *testing.T) {
	var replies []string
	for i := 0; i < 40; i++ {
		replies = append(replies, nativeCall("inspect_fixture", map[string]any{"path": fmt.Sprintf("file-%d", i)}))
	}
	replies = append(replies, nativeCall("inspect_fixture", map[string]any{"path": "final"}), "Inspected")
	g, e, j := &generator{replies: replies}, &executor{}, &journal{}
	r := fixture(g, e, j)
	result, err := r.Run(context.Background(), "inspect files", approve, emit)
	if err != nil || result.Status != runrecord.Answered || e.executed != 41 {
		t.Fatalf("long run: %s %d %v", result.Status, e.executed, err)
	}
}
func TestModelBudgetStopsWithoutReportingSuccess(t *testing.T) {
	g := &generator{replies: []string{nativeCall("inspect_fixture", map[string]any{"path": "fixture"})}, repeat: true}
	e, j := &executor{}, &journal{}
	r := fixture(g, e, j)
	r.Limits = DefaultLimits()
	r.Limits.MaxModelCalls = 2
	result, err := r.Run(context.Background(), "work", approve, emit)
	if err != nil || !strings.Contains(result.Text, "budget") || result.Status != runrecord.Stopped {
		t.Fatalf("stop: %+v %v", result, err)
	}
	if len(g.requests) != 2 || e.prepared != 2 || e.executed != 2 || result.HadToolFailure {
		t.Fatalf("model calls=%d tool preparations=%d executions=%d", len(g.requests), e.prepared, e.executed)
	}
}
func TestRequestsHaveIndependentCallBudgets(t *testing.T) {
	g, e, j := &generator{replies: []string{"first", "second"}}, &executor{}, &journal{}
	r := fixture(g, e, j)
	r.Limits = DefaultLimits()
	r.Limits.MaxModelCalls = 1
	for _, input := range []string{"first request", "second request"} {
		result, err := r.Run(context.Background(), input, approve, emit)
		if err != nil || result.Status != runrecord.Answered || recordedModelCalls(j) != 1 {
			t.Fatalf("request inherited another budget: %+v %v", result, err)
		}
	}
	if len(g.requests) != 2 {
		t.Fatalf("requests=%d", len(g.requests))
	}
}

func recordedModelCalls(j *journal) int {
	for i := len(j.events) - 1; i >= 0; i-- {
		if cp := j.events[i].Checkpoint; cp != nil {
			return cp.ModelCalls
		}
	}
	return 0
}
