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
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/policy"
	"strings"
	"sync"
	"testing"
)

type generator struct {
	purposes          []string
	purposeRequests   []modelapi.Request
	review            func(context.Context, modelapi.Request, func(modelapi.Event) error) error
	mu                sync.Mutex
	reviewRequests    []modelapi.Request
	replies           []string
	requests          []modelapi.Request
	groundingRequests []modelapi.Request
	cancelAt          int
	cancel            context.CancelFunc
	native            bool
}

func (g *generator) Generate(ctx context.Context, r modelapi.Request, emit func(modelapi.Event) error) error {
	if strings.Contains(r.Messages[len(r.Messages)-1].Content, "\nPFC PURPOSE:") {
		g.purposeRequests = append(g.purposeRequests, r)
		reply := "work"
		if len(g.purposes) > 0 {
			reply, g.purposes = g.purposes[0], g.purposes[1:]
		}
		return emit(modelapi.Event{Delta: reply, Done: true})
	}
	if strings.HasPrefix(r.Messages[len(r.Messages)-1].Content, "CONCLAVE REVIEW ") {
		g.mu.Lock()
		g.reviewRequests = append(g.reviewRequests, r)
		g.mu.Unlock()
		if g.review != nil {
			return g.review(ctx, r, emit)
		}
		return emit(modelapi.Event{Delta: "1", Done: true})
	}
	if strings.Contains(r.Messages[len(r.Messages)-1].Content, "\nSELF-GROUNDING:") {
		g.groundingRequests = append(g.groundingRequests, r)
		return emit(modelapi.Event{Delta: "- Evidence: User request and recorded observations; file contents may be unknown.\n- Action: Perform this requested operation.\n- Verify: Inspect the result and run the relevant check.", Done: true})
	}
	g.requests = append(g.requests, r)
	if g.native {
		return emit(modelapi.Event{Done: true, Calls: []toolapi.Call{{ID: "native", Name: "write_file", Arguments: json.RawMessage(`{}`)}}})
	}
	if len(g.requests) > len(g.replies) {
		return fmt.Errorf("script exhausted")
	}
	reply := g.replies[len(g.requests)-1]
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
	path := toolapi.Parameter{Name: "path", Question: "Path?", Kind: toolapi.LiteralPath, Required: true, MaxBytes: 4096}
	return []toolapi.Definition{{Name: "write_file", Description: "Write", Effect: policy.Change, Parameters: []toolapi.Parameter{path, {Name: "content", Question: "Content?", Kind: toolapi.Text, Required: true, AllowEmpty: true, MaxBytes: 4096}}}, {Name: "read_file", Description: "Read", Effect: policy.Read, Parameters: []toolapi.Parameter{path}}, {Name: "execute_command", Description: "Run", Effect: policy.Command, Parameters: []toolapi.Parameter{{Name: "command", Question: "Command?", Kind: toolapi.Text, Required: true, MaxBytes: 4096}}}}
}
func (e *executor) Prepare(_ context.Context, _ policy.Settings, _ bool, c toolapi.Call) (toolapi.Execution, error) {
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
		e.outcomes = e.outcomes[1:]
		return result, nil
	}
	return toolapi.Result{Content: json.RawMessage(`{"ok":true}`), Summary: "ok", Process: &toolapi.ProcessOutcome{Status: "exited", ExitCode: 0}}, nil
}

type journal struct {
	failKind string
	messages []modelapi.Message
	events   []Event
}

func (j *journal) ContextMessages(_ context.Context, _ string, after int64, maxBytes int) ([]modelapi.Message, error) {
	if after != 0 {
		return nil, fmt.Errorf("test journal does not support compaction")
	}
	bytes := 0
	for _, m := range j.messages {
		bytes += len(m.Content) + len(m.Calls)
	}
	if bytes > maxBytes {
		return nil, fmt.Errorf("conversation limit exceeded")
	}
	return append([]modelapi.Message(nil), j.messages...), nil
}
func (j *journal) AppendMessage(_ context.Context, _ string, m modelapi.Message) error {
	j.messages = append(j.messages, m)
	return nil
}
func (j *journal) RecordEvent(_ context.Context, _ string, e Event) error {
	if j.failKind != "" && e.Kind == j.failKind {
		return fmt.Errorf("simulated journal failure")
	}
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
func emit(Event) error                                        { return nil }
func approve(context.Context, toolapi.Approval) (bool, error) { return true, nil }
func TestDeniedToolStopsAndPersistsStructuredResult(t *testing.T) {
	g, e, j := &generator{replies: []string{`{"tool":"write_file"}`, "file.txt", "new"}}, &executor{approval: true}, &journal{}
	r := fixture(g, e, j)
	result, err := r.Run(context.Background(), "change", false, func(context.Context, toolapi.Approval) (bool, error) { return false, nil }, emit)
	if !errors.Is(err, errApprovalDenied) || e.executed != 0 || e.prepared != 1 || result.Status != Stopped || !result.HadToolFailure {
		t.Fatalf("denial: %+v %v", result, err)
	}
	if len(g.requests) != 3 || len(j.messages) != 4 || j.messages[3].Role != "tool" {
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
	g := &generator{replies: []string{`{"tool":"write_file"}`, "file.txt", "unfinished"}, cancelAt: 3, cancel: cancel}
	e, j := &executor{}, &journal{}
	r := fixture(g, e, j)
	result, err := r.Run(ctx, "change", false, approve, emit)
	if !errors.Is(err, context.Canceled) || e.prepared != 0 || e.executed != 0 || result.Status != Interrupted {
		t.Fatalf("partial argument: %+v %v", result, err)
	}
	var last *Checkpoint
	for _, event := range j.events {
		if event.Checkpoint != nil {
			last = event.Checkpoint
		}
	}
	if last == nil || last.Status != Interrupted || string(last.Arguments["path"]) != `"file.txt"` {
		t.Fatalf("checkpoint: %+v", last)
	}
}
func TestChangesRequireARealCheckAfterLatestMutation(t *testing.T) {
	g := &generator{purposes: []string{"check"}, replies: []string{`{"tool":"write_file"}`, "file.txt", "new", "DONE\nTests passed", `{"tool":"read_file"}`, "file.txt", "DONE\nAll good", "WAIT\nAn approved test command or user review is needed"}}
	e, j := &executor{}, &journal{}
	r := fixture(g, e, j)
	result, err := r.Run(context.Background(), "change", false, approve, emit)
	if err != nil || result.Status != NeedsInput || result.Evidence != "" {
		t.Fatalf("unchecked completion: %+v %v", result, err)
	}
}
func TestVerifiedCommandAllowsCompletionAndFullContexts(t *testing.T) {
	g := &generator{purposes: []string{"check"}, replies: []string{`{"tool":"write_file"}`, "file.txt", "new", `{"tool":"execute_command"}`, "go test ./...", "DONE\nThe check passed"}}
	e, j := &executor{}, &journal{}
	r := fixture(g, e, j)
	r.Settings.Scope = policy.System
	result, err := r.Run(context.Background(), "change", false, approve, emit)
	if err != nil || result.Status != Verified || !strings.Contains(result.Evidence, "go test") {
		t.Fatalf("completion: %+v %v", result, err)
	}
	for _, request := range g.requests {
		if len(request.Messages) < 3 || !strings.Contains(request.Messages[0].Content, r.ProjectInstructions) {
			t.Fatal("rules or full context lost")
		}
	}
}
func TestRalphLoopCanContinueBeyondThirtyTwoActions(t *testing.T) {
	var replies []string
	for i := 0; i < 40; i++ {
		replies = append(replies, `{"tool":"read_file"}`, fmt.Sprintf("file-%d", i))
	}
	replies = append(replies, `{"tool":"read_file"}`, "final", "DONE\nInspected")
	g, e, j := &generator{purposes: strings.Split(strings.Repeat("work|", 40)+"check", "|"), replies: replies}, &executor{}, &journal{}
	r := fixture(g, e, j)
	result, err := r.Run(context.Background(), "inspect files", false, approve, emit)
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
	}{{strings.Split(strings.Repeat("CONTINUE\ntry|", 8)+"CONTINUE\ntry", "|"), false, 128, "stalled"}, {[]string{`{"tool":"read_file"}`, "file"}, false, 1, "budget"}, {nil, true, 128, "native JSON"}} {
		g, e, j := &generator{replies: tc.replies, native: tc.native}, &executor{}, &journal{}
		r := fixture(g, e, j)
		r.Limits = DefaultLimits()
		r.Limits.MaxIterations = tc.limit
		result, err := r.Run(context.Background(), "work", false, approve, emit)
		if err == nil || !strings.Contains(err.Error(), tc.want) || result.Status != Stopped {
			t.Fatalf("stop: %+v %v", result, err)
		}
	}
}
func TestPlanMutationRejectedBeforeArgumentCollection(t *testing.T) {
	g, e, j := &generator{replies: []string{`{"tool":"write_file"}`}}, &executor{}, &journal{}
	r := fixture(g, e, j)
	_, err := r.Run(context.Background(), "plan", true, approve, emit)
	if err == nil || e.prepared != 0 || len(g.requests) != 1 {
		t.Fatalf("planning mutation: %v", err)
	}
}

func TestApprovedStepsShareOneCallBudget(t *testing.T) {
	usage := &Usage{}
	g, e, j := &generator{purposes: []string{"check"}, replies: []string{`{"tool":"read_file"}`, "file", "DONE\nfirst", "DONE\nsecond"}}, &executor{}, &journal{}
	r := fixture(g, e, j)
	r.Usage = usage
	r.Limits = DefaultLimits()
	r.Limits.MaxModelCalls = 5
	if _, err := r.Run(context.Background(), "step one", false, approve, emit); err != nil {
		t.Fatal(err)
	}
	result, err := r.Run(context.Background(), "step two", false, approve, emit)
	if err == nil || result.Status != Stopped || len(g.requests) != 3 {
		t.Fatalf("budget reset between steps: %+v %v", result, err)
	}
}
