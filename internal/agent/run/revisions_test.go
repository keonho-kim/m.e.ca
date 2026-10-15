// Purpose: Verify repeated edits need no check gate and file cycles survive interleaved reads.
// Pattern: Scripted runtime behavior tests.
// Usage: go test ./internal/agent/run -run 'TestRepeatedEdits|TestFileCycle'.
// Related: internal/agent/run/observation.go, internal/agent/run/action.go.
package run

import (
	"context"
	"encoding/json"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"strings"
	"testing"
)

func changedFile(before, after string) toolapi.Result {
	return toolapi.Result{Content: json.RawMessage(`{"changed":true}`), Summary: "changed", Changes: []toolapi.FileChange{{Path: "a", BeforeDigest: before, AfterDigest: after}}}
}

func TestRepeatedEditsDoNotRequireCheck(t *testing.T) {
	var replies []string
	var outcomes []toolapi.Result
	for _, pair := range [][2]string{{"0", "1"}, {"1", "2"}, {"2", "3"}, {"3", "4"}} {
		replies = append(replies, nativeCall("write_file", map[string]any{"path": "a", "content": pair[1]}))
		outcomes = append(outcomes, changedFile(pair[0], pair[1]))
	}
	replies = append(replies, "Changed a; no tests run")
	g, e := &generator{replies: replies}, &executor{outcomes: outcomes}
	result, err := fixture(g, e, &journal{}).Run(context.Background(), "repair a", false, approve, emit)
	if err != nil || result.Status != Answered || e.executed != 4 || len(g.requests) != 5 {
		t.Fatalf("%+v executions=%d requests=%d %v", result, e.executed, len(g.requests), err)
	}
}

func TestFileCycleRequestsRecoveryAcrossReads(t *testing.T) {
	write := func(value string) string {
		return nativeCall("write_file", map[string]any{"path": "a", "content": value})
	}
	g := &generator{replies: []string{write("second"), nativeCall("read_file", map[string]any{"path": "a"}), write("first"), nativeCall("execute_command", map[string]any{"command": "verify"}), "Done"}}
	e := &executor{outcomes: []toolapi.Result{changedFile("first", "second"), {Content: json.RawMessage(`{"content":"second"}`)}, changedFile("second", "first"), {Content: json.RawMessage(`{"ok":true}`), Process: &toolapi.ProcessOutcome{Status: "exited", ExitCode: 0}}}}
	j := &journal{}
	r := fixture(g, e, j)
	result, err := r.Run(context.Background(), "repair a", false, approve, emit)
	if err != nil || result.Status != Answered {
		t.Fatalf("%+v %v", result, err)
	}
	for _, event := range j.events {
		if event.Kind == "recovery" && strings.Contains(event.Text, "previously observed state") {
			return
		}
	}
	t.Fatal("read erased file cycle")
}
