// Purpose: Verify repeated edits pause for checks and file cycles survive interleaved reads.
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

func TestRepeatedEditsRequireCheckThenAllowCorrection(t *testing.T) {
	write := func(value string) string {
		return nativeCall("write_file", map[string]any{"path": "a", "content": value})
	}
	g := &generator{replies: []string{write("1"), write("2"), write("3"), write("4"), nativeCall("execute_command", map[string]any{"command": "verify", "purpose": "check"}), write("4"), nativeCall("execute_command", map[string]any{"command": "verify", "purpose": "check"}), "Done"}}
	e := &executor{outcomes: []toolapi.Result{changedFile("0", "1"), changedFile("1", "2"), changedFile("2", "3"), {Content: json.RawMessage(`{"failed":true}`), Failed: true, Process: &toolapi.ProcessOutcome{Status: "exited", ExitCode: 1}}, changedFile("3", "4"), {Content: json.RawMessage(`{"ok":true}`), Process: &toolapi.ProcessOutcome{Status: "exited", ExitCode: 0}}}}
	j := &journal{}
	r := fixture(g, e, j)
	result, err := r.Run(context.Background(), "repair a", false, approve, emit)
	if err != nil || result.Status != Verified || e.executed != 6 {
		t.Fatalf("result=%+v executed=%d err=%v", result, e.executed, err)
	}
	found := false
	for _, m := range j.messages {
		if m.Role == "tool" && strings.Contains(m.Content, "Verification checkpoint required") {
			found = true
		}
	}
	if !found {
		t.Fatal("fourth edit was not blocked")
	}
	request := g.requests[len(g.requests)-1]
	if !strings.Contains(request.Messages[len(request.Messages)-1].Content, "current change revision has passed") {
		t.Fatal("completion guidance missing")
	}
}

func TestFileCycleRequestsRecoveryAcrossReads(t *testing.T) {
	write := func(value string) string {
		return nativeCall("write_file", map[string]any{"path": "a", "content": value})
	}
	g := &generator{replies: []string{write("second"), nativeCall("read_file", map[string]any{"path": "a"}), write("first"), nativeCall("execute_command", map[string]any{"command": "verify", "purpose": "check"}), "Done"}}
	e := &executor{outcomes: []toolapi.Result{changedFile("first", "second"), {Content: json.RawMessage(`{"content":"second"}`)}, changedFile("second", "first"), {Content: json.RawMessage(`{"ok":true}`), Process: &toolapi.ProcessOutcome{Status: "exited", ExitCode: 0}}}}
	j := &journal{}
	r := fixture(g, e, j)
	result, err := r.Run(context.Background(), "repair a", false, approve, emit)
	if err != nil || result.Status != Verified {
		t.Fatalf("%+v %v", result, err)
	}
	for _, event := range j.events {
		if event.Kind == "recovery" && strings.Contains(event.Text, "previously observed state") {
			return
		}
	}
	t.Fatal("read erased file cycle")
}
