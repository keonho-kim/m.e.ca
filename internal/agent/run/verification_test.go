// Purpose: Verify command failures remain observations and stale changes can be retried.
// Pattern: Task completion integration tests.
// Usage: go test ./internal/agent/run -run TestCompletion.
// Related: internal/agent/run/action.go, internal/agent/run/observations.go.
package run

import (
	"context"
	"encoding/json"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/policy"
	"strings"
	"testing"
)

func TestCompletionRetainsFailedCommandHistory(t *testing.T) {
	g := &generator{replies: []string{nativeCall("execute_command", map[string]any{"command": "test-a"}), nativeCall("execute_command", map[string]any{"command": "test-b"}), "Test-a failed; test-b passed"}}
	e := &executor{outcomes: []toolapi.Result{{Content: []byte(`{"ExitCode":1}`), Failed: true, Process: &toolapi.ProcessOutcome{Status: "exited", ExitCode: 1}}, {Content: []byte(`{"ExitCode":0}`), Process: &toolapi.ProcessOutcome{Status: "exited", ExitCode: 0}}}}
	j := &journal{}
	r := fixture(g, e, j)
	r.Settings.Scope = policy.System
	result, err := r.Run(context.Background(), "verify both checks", approve, emit)
	if err != nil || result.Status != Answered {
		t.Fatalf("premature completion: %+v %v", result, err)
	}
	body, err := json.Marshal(g.requests[len(g.requests)-1].Messages)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "test-a") || !strings.Contains(string(body), "ExitCode") {
		t.Fatal("failed criterion lost")
	}
}

func TestStaleGuardAllowsFreshPreparationOfSameEdit(t *testing.T) {
	g := &generator{replies: []string{nativeCall("write_file", map[string]any{"path": "a", "content": "new"}), nativeCall("write_file", map[string]any{"path": "a", "content": "new"}), nativeCall("execute_command", map[string]any{"command": "verify-a"}), "Verified"}}
	e := &executor{outcomes: []toolapi.Result{
		{Failed: true, Failure: &toolapi.Failure{Code: toolapi.StaleTarget, Detail: "file changed before execution", Next: "Inspect latest version and prepare again."}},
		{Content: []byte(`{"changed":true}`)},
		{Content: []byte(`{"ExitCode":0}`), Process: &toolapi.ProcessOutcome{Status: "exited", ExitCode: 0}},
	}}
	r := fixture(g, e, &journal{})
	r.Settings.Scope = policy.System
	result, err := r.Run(context.Background(), "update a", approve, emit)
	if err != nil || result.Status != Answered || e.executed != 3 {
		t.Fatalf("safe reprepare blocked: %+v executions=%d err=%v", result, e.executed, err)
	}
}
