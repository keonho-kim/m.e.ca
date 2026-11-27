// Purpose: Supply deterministic model replies and native tool effects for runtime behavior tests.
// Pattern: Package-owned dependency fixtures.
// Usage: Runtime tests use fixture and select explicit replies, outcomes and approval behavior.
// Related: internal/agent/run/native_fixture_test.go, internal/agent/run/journal_fixture_test.go, internal/agent/run/run_test.go.
package run

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/policy"
)

type generator struct {
	replies  []string
	requests []modelapi.Request
	cancelAt int
	cancel   context.CancelFunc
	repeat   bool
}

func (g *generator) Generate(ctx context.Context, r modelapi.Request, emit func(modelapi.Event) error) error {
	g.requests = append(g.requests, r)
	index := len(g.requests) - 1
	if g.repeat && len(g.replies) > 0 {
		index %= len(g.replies)
	}
	if index >= len(g.replies) {
		return fmt.Errorf("script exhausted")
	}
	reply := g.replies[index]
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
