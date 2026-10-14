// Purpose: Verify DONE cannot hide failed checks behind an unrelated successful command.
// Pattern: Task completion integration tests.
// Usage: go test ./internal/agent/run -run TestCompletion.
// Related: internal/agent/run/action.go, internal/agent/verification/state.go.
package run

import (
	"context"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/policy"
	"strings"
	"testing"
)

func TestCompletionKeepsFailedChecksPending(t *testing.T) {
	g := &generator{purposes: []string{"check", "check"}, replies: []string{`{"tool":"execute_command"}`, "test-a", `{"tool":"execute_command"}`, "test-b", "DONE\nEverything passed", "WAIT\nTest-a still needs repair"}}
	e := &executor{outcomes: []toolapi.Result{{Content: []byte(`{"ExitCode":1}`), Failed: true, Process: &toolapi.ProcessOutcome{Status: "exited", ExitCode: 1}}, {Content: []byte(`{"ExitCode":0}`), Process: &toolapi.ProcessOutcome{Status: "exited", ExitCode: 0}}}}
	j := &journal{}
	r := fixture(g, e, j)
	r.Settings.Scope = policy.System
	result, err := r.Run(context.Background(), "verify both checks", false, approve, emit)
	if err != nil || result.Status != NeedsInput || result.Evidence != "" {
		t.Fatalf("premature completion: %+v %v", result, err)
	}
	facts := g.requests[len(g.requests)-1].Messages
	if !strings.Contains(facts[len(facts)-1].Content, "test-a") || !strings.Contains(facts[len(facts)-1].Content, "failed") {
		t.Fatal("failed criterion lost")
	}
}

func TestStaleGuardAllowsFreshPreparationOfSameEdit(t *testing.T) {
	g := &generator{purposes: []string{"check"}, replies: []string{`{"tool":"write_file"}`, "a", "new", `{"tool":"write_file"}`, "a", "new", `{"tool":"execute_command"}`, "verify-a", "DONE\nVerified"}}
	e := &executor{outcomes: []toolapi.Result{
		{Failed: true, Failure: &toolapi.Failure{Code: toolapi.StaleTarget, Detail: "file changed before execution", Next: "Inspect latest version and prepare again."}},
		{Content: []byte(`{"changed":true}`)},
		{Content: []byte(`{"ExitCode":0}`), Process: &toolapi.ProcessOutcome{Status: "exited", ExitCode: 0}},
	}}
	r := fixture(g, e, &journal{})
	r.Settings.Scope = policy.System
	result, err := r.Run(context.Background(), "update a", false, approve, emit)
	if err != nil || result.Status != Verified || e.executed != 3 {
		t.Fatalf("safe reprepare blocked: %+v executions=%d err=%v", result, e.executed, err)
	}
}
