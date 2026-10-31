// Purpose: Verify native tool execution, model-call budgets and final reporting.
// Pattern: Task behavior tests with scripted model replies.
// Usage: go test ./internal/agent/run.
// Related: internal/agent/run/run.go, internal/agent/run/action.go.
package run

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/policy"
	"strings"
	"testing"
)

type generator struct {
	replies  []string
	requests []modelapi.Request
	cancelAt int
	cancel   context.CancelFunc
	native   bool
}

func (g *generator) Generate(ctx context.Context, r modelapi.Request, emit func(modelapi.Event) error) error {
	g.requests = append(g.requests, r)
	if g.native {
		return emit(modelapi.Event{Done: true, Calls: []toolapi.Call{{ID: "native", Name: "write_file", Arguments: json.RawMessage(`{}`)}}})
	}
	if len(g.requests) > len(g.replies) {
		return fmt.Errorf("script exhausted")
	}
	reply := g.replies[len(g.requests)-1]
	if strings.HasPrefix(reply, "native:") {
		var call toolapi.Call
		if err := json.Unmarshal([]byte(strings.TrimPrefix(reply, "native:")), &call); err != nil {
			return err
		}
		call.ID = fmt.Sprintf("call_%d", len(g.requests))
		if g.cancelAt == len(g.requests) {
			g.cancel()
			return ctx.Err()
		}
		return emit(modelapi.Event{Calls: []toolapi.Call{call}, Done: true})
	}
	if err := emit(modelapi.Event{Delta: reply}); err != nil {
		return err
	}
	if g.cancelAt == len(g.requests) {
		g.cancel()
		return ctx.Err()
	}
	return emit(modelapi.Event{Done: true})
}

type executor struct {
	prepared, executed int
	outcomes           []toolapi.Result
	calls              []toolapi.Call
	approval           bool
}

func (*executor) Definitions() []toolapi.Definition {
	path := toolapi.Parameter{Name: "path", Description: "Path?", Kind: toolapi.LiteralPath, Required: true, MaxBytes: 4096}
	definitions := []toolapi.Definition{{Name: "write_file", Description: "Write", Effect: policy.Change, Parameters: []toolapi.Parameter{path, {Name: "content", Description: "Content?", Kind: toolapi.Text, Required: true, AllowEmpty: true, MaxBytes: 4096}}}, {Name: "inspect_fixture", Description: "Read", Effect: policy.Read, Parameters: []toolapi.Parameter{path}}, {Name: "execute_command", Description: "Run", Effect: policy.Command, Parameters: []toolapi.Parameter{{Name: "command", Description: "Command?", Kind: toolapi.Text, Required: true, MaxBytes: 4096}}}}
	return definitions
}
func (e *executor) Prepare(_ context.Context, _ policy.Settings, c toolapi.Call) (toolapi.Execution, error) {
	e.prepared++
	e.calls = append(e.calls, c)
	return e, nil
}
func (e *executor) Approval() toolapi.Approval {
	approval := toolapi.Approval{ID: "op", Required: e.approval}
	if len(e.calls) > 0 {
		call := e.calls[len(e.calls)-1]
		if call.Name == "write_file" {
			var args struct{ Path, Content string }
			if err := json.Unmarshal(call.Arguments, &args); err != nil {
				panic(err)
			}
			lines := 0
			if args.Content != "" {
				lines = 1
			}
			candidate := toolapi.BindCandidate(call, []toolapi.FileDelta{{Path: args.Path, Added: lines, Diff: "+" + args.Content}})
			approval.Candidate = &candidate
		}
	}
	return approval
}
func (e *executor) Execute(context.Context, bool) (toolapi.Result, error) {
	e.executed++
	if len(e.outcomes) > 0 {
		result := e.outcomes[0]
		result.Started = true
		e.outcomes = e.outcomes[1:]
		return result, nil
	}
	if len(e.calls) > 0 && e.calls[len(e.calls)-1].Name == "write_file" {
		var fields struct{ Path, Content string }
		if err := json.Unmarshal(e.calls[len(e.calls)-1].Arguments, &fields); err != nil {
			return toolapi.Result{}, err
		}
		return toolapi.Result{Content: json.RawMessage(`{"changed":true}`), Summary: "ok", Target: fields.Path, Versions: map[string]string{fields.Path: fields.Content}, Changes: []toolapi.FileChange{{Path: fields.Path, BeforeDigest: "missing", AfterDigest: fields.Content}}}, nil
	}
	return toolapi.Result{Started: true, Content: json.RawMessage(`{"ok":true}`), Summary: "ok", Process: &toolapi.ProcessOutcome{Status: "exited", ExitCode: 0}}, nil
}

func fixture(g *generator, e *executor, j *journal) Runner {
	return Runner{Generator: g, Tools: e, Journal: j, Session: "s", Model: "m", ProjectInstructions: "Use focused tests. Scope cannot be bypassed.", Settings: policy.Settings{Scope: policy.Directory}, Redact: func(s string) string { return s }}
}
func emit(runrecord.Event) error                              { return nil }
func approve(context.Context, toolapi.Approval) (bool, error) { return true, nil }
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
	for _, tc := range []struct {
		replies []string
		native  bool
		limit   int
		want    string
	}{{[]string{nativeCall("inspect_fixture", map[string]any{"path": "file"})}, false, 1, "budget"}, {nil, true, 2, "budget"}} {
		g, e, j := &generator{replies: tc.replies, native: tc.native}, &executor{}, &journal{}
		r := fixture(g, e, j)
		r.Limits = DefaultLimits()
		r.Limits.MaxModelCalls = tc.limit
		result, err := r.Run(context.Background(), "work", approve, emit)
		if err != nil || !strings.Contains(result.Text, tc.want) || result.Status != runrecord.Stopped {
			t.Fatalf("stop: %+v %v", result, err)
		}
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
