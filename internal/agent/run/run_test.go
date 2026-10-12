// Purpose: Verify staged tool execution, bounded Ralph progress and evidence-gated completion.
// Pattern: Task behavior tests with scripted model replies.
// Usage: go test ./internal/agent/run.
// Related: internal/agent/run/run.go, internal/agent/run/action.go.
package run

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent"
	"github.com/keonho-kim/v-code/internal/policy"
	"strings"
	"testing"
)

type generator struct {
	replies  []string
	requests []agent.ModelRequest
	cancelAt int
	cancel   context.CancelFunc
	native   bool
}

func (g *generator) Generate(ctx context.Context, r agent.ModelRequest, emit func(agent.ModelEvent) error) error {
	g.requests = append(g.requests, r)
	if g.native {
		return emit(agent.ModelEvent{Done: true, Calls: []agent.ToolCall{{ID: "native", Name: "write_file", Arguments: json.RawMessage(`{}`)}}})
	}
	if len(g.requests) > len(g.replies) {
		return fmt.Errorf("script exhausted")
	}
	reply := g.replies[len(g.requests)-1]
	if err := emit(agent.ModelEvent{Delta: reply}); err != nil {
		return err
	}
	if g.cancelAt == len(g.requests) {
		g.cancel()
		return ctx.Err()
	}
	return emit(agent.ModelEvent{Done: true})
}

type executor struct {
	prepared, executed int
	calls              []agent.ToolCall
	approval           bool
}

func (*executor) Definitions() []agent.ToolDefinition {
	path := agent.ToolParameter{Name: "path", Question: "Path?", Kind: agent.Text, Required: true, MaxBytes: 4096}
	return []agent.ToolDefinition{{Name: "write_file", Description: "Write", Effect: policy.Change, Parameters: []agent.ToolParameter{path, {Name: "content", Question: "Content?", Kind: agent.Text, Required: true, AllowEmpty: true, MaxBytes: 4096}}}, {Name: "read_file", Description: "Read", Effect: policy.Read, Parameters: []agent.ToolParameter{path}}, {Name: "execute_command", Description: "Run", Effect: policy.Command, Parameters: []agent.ToolParameter{{Name: "command", Question: "Command?", Kind: agent.Text, Required: true, MaxBytes: 4096}}}}
}
func (e *executor) Prepare(_ context.Context, _ policy.Settings, _ bool, c agent.ToolCall) (agent.ToolExecution, error) {
	e.prepared++
	e.calls = append(e.calls, c)
	return e, nil
}
func (e *executor) Approval() agent.ApprovalRequest {
	return agent.ApprovalRequest{ID: "op", Required: e.approval}
}
func (e *executor) Execute(context.Context, bool) (agent.ToolResult, error) {
	e.executed++
	return agent.ToolResult{Content: json.RawMessage(`{"ok":true}`), Summary: "ok"}, nil
}

type journal struct {
	messages []agent.Message
	events   []Event
}

func (j *journal) AppendMessage(_ context.Context, _ string, m agent.Message) error {
	j.messages = append(j.messages, m)
	return nil
}
func (j *journal) RecordEvent(_ context.Context, _ string, e Event) error {
	body, err := json.Marshal(e)
	if err != nil {
		return err
	}
	var copy Event
	if err = json.Unmarshal(body, &copy); err != nil {
		return err
	}
	j.events = append(j.events, copy)
	return nil
}
func fixture(g *generator, e *executor, j *journal) Runner {
	return Runner{Generator: g, Tools: e, Journal: j, Session: "s", Model: "m", ProjectInstructions: "Use focused tests. Scope cannot be bypassed.", Settings: policy.Settings{Permission: policy.Approve, Scope: policy.Directory}, Redact: func(s string) string { return s }}
}
func emit(Event) error                                             { return nil }
func approve(context.Context, agent.ApprovalRequest) (bool, error) { return true, nil }
func TestDeniedToolStopsAndPersistsStructuredResult(t *testing.T) {
	g, e, j := &generator{replies: []string{"TOOL write_file", "file.txt", "new"}}, &executor{approval: true}, &journal{}
	r := fixture(g, e, j)
	result, err := r.Run(context.Background(), nil, "change", false, func(context.Context, agent.ApprovalRequest) (bool, error) { return false, nil }, emit)
	if !errors.Is(err, errApprovalDenied) || e.executed != 0 || e.prepared != 1 || result.Status != Stopped || !result.HadToolFailure {
		t.Fatalf("denial: %+v %v", result, err)
	}
	if len(g.requests) != 3 || len(j.messages) != 3 || j.messages[2].Role != "tool" {
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
	g := &generator{replies: []string{"TOOL write_file", "file.txt", "unfinished"}, cancelAt: 3, cancel: cancel}
	e, j := &executor{}, &journal{}
	r := fixture(g, e, j)
	result, err := r.Run(ctx, nil, "change", false, approve, emit)
	if !errors.Is(err, context.Canceled) || e.prepared != 0 || e.executed != 0 || result.Status != Interrupted {
		t.Fatalf("partial argument: %+v %v", result, err)
	}
	last := j.events[len(j.events)-1].Checkpoint
	if last == nil || last.Status != Interrupted || string(last.Arguments["path"]) != `"file.txt"` {
		t.Fatalf("checkpoint: %+v", last)
	}
}
func TestChangesRequireARealCheckAfterLatestMutation(t *testing.T) {
	g := &generator{replies: []string{"TOOL write_file", "file.txt", "new", "DONE\nTests passed", "CHECK read_file", "file.txt", "DONE\nAll good", "WAIT\nAn approved test command or user review is needed"}}
	e, j := &executor{}, &journal{}
	r := fixture(g, e, j)
	result, err := r.Run(context.Background(), nil, "change", false, approve, emit)
	if err != nil || result.Status != NeedsInput || result.Evidence != "" {
		t.Fatalf("unchecked completion: %+v %v", result, err)
	}
}
func TestVerifiedCommandAllowsCompletionAndFreshContexts(t *testing.T) {
	g := &generator{replies: []string{"TOOL write_file", "file.txt", "new", "CHECK execute_command", "go test ./...", "DONE\nThe check passed"}}
	e, j := &executor{}, &journal{}
	r := fixture(g, e, j)
	r.Settings.Scope = policy.System
	result, err := r.Run(context.Background(), nil, "change", false, approve, emit)
	if err != nil || result.Status != Verified || !strings.Contains(result.Evidence, "go test") {
		t.Fatalf("completion: %+v %v", result, err)
	}
	for _, request := range g.requests {
		if len(request.Messages) != 2 || !strings.Contains(request.Messages[0].Content, r.ProjectInstructions) {
			t.Fatal("rules or fresh context lost")
		}
	}
}
func TestRalphLoopCanContinueBeyondThirtyTwoActions(t *testing.T) {
	var replies []string
	for i := 0; i < 40; i++ {
		replies = append(replies, "TOOL read_file", fmt.Sprintf("file-%d", i))
	}
	replies = append(replies, "CHECK read_file", "final", "DONE\nInspected")
	g, e, j := &generator{replies: replies}, &executor{}, &journal{}
	r := fixture(g, e, j)
	result, err := r.Run(context.Background(), nil, "inspect files", false, approve, emit)
	if err != nil || result.Status != Verified || e.executed != 41 {
		t.Fatalf("long run: %s %d %v", result.Status, e.executed, err)
	}
}
func TestBudgetStallAndNativeCallsNeverReportSuccess(t *testing.T) {
	for _, tc := range []struct {
		replies []string
		native  bool
		limit   int
		want    string
	}{{[]string{"CONTINUE\ntry", "CONTINUE\ntry", "CONTINUE\ntry"}, false, 128, "stalled"}, {[]string{"TOOL read_file", "file"}, false, 1, "budget"}, {nil, true, 128, "native JSON"}} {
		g, e, j := &generator{replies: tc.replies, native: tc.native}, &executor{}, &journal{}
		r := fixture(g, e, j)
		r.Limits = DefaultLimits()
		r.Limits.MaxIterations = tc.limit
		result, err := r.Run(context.Background(), nil, "work", false, approve, emit)
		if err == nil || !strings.Contains(err.Error(), tc.want) || result.Status != Stopped {
			t.Fatalf("stop: %+v %v", result, err)
		}
	}
}
func TestPlanMutationRejectedBeforeArgumentCollection(t *testing.T) {
	g, e, j := &generator{replies: []string{"TOOL write_file"}}, &executor{}, &journal{}
	r := fixture(g, e, j)
	_, err := r.Run(context.Background(), nil, "plan", true, approve, emit)
	if err == nil || e.prepared != 0 || len(g.requests) != 1 {
		t.Fatalf("planning mutation: %v", err)
	}
}

func TestApprovedStepsShareOneCallBudget(t *testing.T) {
	usage := &Usage{}
	g, e, j := &generator{replies: []string{"CHECK read_file", "file", "DONE\nfirst", "DONE\nsecond"}}, &executor{}, &journal{}
	r := fixture(g, e, j)
	r.Usage = usage
	r.Limits = DefaultLimits()
	r.Limits.MaxModelCalls = 3
	if _, err := r.Run(context.Background(), nil, "step one", false, approve, emit); err != nil {
		t.Fatal(err)
	}
	result, err := r.Run(context.Background(), nil, "step two", false, approve, emit)
	if err == nil || result.Status != Stopped || len(g.requests) != 3 {
		t.Fatalf("budget reset between steps: %+v %v", result, err)
	}
}
