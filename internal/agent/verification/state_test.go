// Purpose: Verify checks require observed process completion and remain pending across unrelated passes or edits.
// Pattern: State transition tests.
// Usage: go test ./internal/agent/verification.
// Related: internal/agent/verification/state.go, internal/agent/run/action.go.
package verification

import (
	"github.com/keonho-kim/v-code/internal/agent/tool"
	"testing"
)

func TestFailedCheckRequiresSameCheckAtCurrentRevision(t *testing.T) {
	call := tool.Call{ID: "c", Name: "execute_command", Arguments: []byte(`{"command":"test-a"}`)}
	records, err := Observe(nil, call, tool.Result{Started: true, Failed: true, Process: &tool.ProcessOutcome{Status: "exited", ExitCode: 1}}, 1)
	if err != nil {
		t.Fatal(err)
	}
	call.ID = "d"
	call.Arguments = []byte(`{"command":"test-b"}`)
	records, err = Observe(records, call, tool.Result{Started: true, Process: &tool.ProcessOutcome{Status: "exited", ExitCode: 0}}, 1)
	if err != nil || len(Pending(records, 1)) != 1 {
		t.Fatalf("unrelated pass cleared failure: %+v %v", records, err)
	}
	call.Arguments = []byte(`{"command":"test-a"}`)
	records, err = Observe(records, call, tool.Result{Started: true, Process: &tool.ProcessOutcome{Status: "exited", ExitCode: 0}}, 1)
	if err != nil || len(Pending(records, 1)) != 0 || len(Pending(records, 2)) != 2 {
		t.Fatalf("revision gate=%+v %v", records, err)
	}
}

func TestMissingProcessStatusNeverPasses(t *testing.T) {
	records, err := Observe(nil, tool.Call{ID: "c", Name: "execute_command", Arguments: []byte(`{"command":"test"}`)}, tool.Result{Started: true}, 0)
	if err != nil || len(Pending(records, 0)) != 1 {
		t.Fatalf("missing status passed: %+v %v", records, err)
	}
}

func TestPreparationFailureDoesNotCreateRequiredCheck(t *testing.T) {
	records, err := Observe(nil, tool.Call{ID: "c", Name: "execute_command", Arguments: []byte(`{"command":"missing"}`)}, tool.Result{Failed: true}, 0)
	if err != nil || len(records) != 0 {
		t.Fatalf("unstarted check became obligation: %+v %v", records, err)
	}
}
