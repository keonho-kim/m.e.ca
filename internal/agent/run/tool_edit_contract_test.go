// Purpose: Verify ordinary edits and the recovery of invalid edit metadata.
// Pattern: Scripted filesystem integration test.
// Usage: go test ./internal/agent/run -run TestToolReadThenEdit.
// Related: internal/tools/local/prepare.go, internal/agent/run/action_result.go.
package run

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func TestToolReadThenEditIsAutomatic(t *testing.T) {
	r, _, root, _, _ := realBatchRunner(t, nil)
	path := filepath.Join(root, "note.txt")
	if err := os.WriteFile(path, []byte("name: demo\nstatus: pending\ncount: 3\n"), 0600); err != nil {
		t.Fatal(err)
	}
	g := &generator{replies: []string{nativeCall("execute_command", map[string]any{"reason": "Read or update known source", "command": "cat note.txt"}), "missing read result", "Updated the requested second line."}}
	r.Generator = g
	approvals := 0
	result, err := r.Run(context.Background(), "Read note.txt and change only status: pending to status: done.", func(_ context.Context, proposal toolapi.Approval) (bool, error) {
		approvals++
		if proposal.Candidate != nil && proposal.Candidate.Lines() != 2 {
			t.Fatal("approval not bound to actual edit")
		}
		return true, nil
	}, func(e runrecord.Event) error {
		if e.Kind == "tool_succeeded" && e.Call != nil && e.Call.Name == "execute_command" {
			var read struct{ Stdout string }
			if err := json.Unmarshal(e.Output, &read); err != nil {
				return err
			}
			old := strings.SplitAfter(read.Stdout, "\n")[1]
			g.replies[1] = nativeCall("edit_file", map[string]any{"reason": "Read or update known source", "path": "note.txt", "edits": []map[string]any{{"old_text": old, "new_text": "status: done"}}})
		}
		return nil
	})
	if err != nil || result.Status != runrecord.Answered {
		t.Fatalf("result=%+v error=%v", result, err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "name: demo\nstatus: done\ncount: 3\n" {
		t.Fatalf("content=%q error=%v", got, err)
	}
	if approvals != 0 {
		t.Fatalf("ordinary operations requested %d approvals", approvals)
	}
}

func TestMissingEditReasonReturnsPairedFeedback(t *testing.T) {
	r, _, root, journal, _ := realBatchRunner(t, nil)
	const observed = "inspection evidence for the edit"
	read := toolapi.Call{ID: "inspect", Name: toolapi.BrowserInspect, Arguments: json.RawMessage(`{"view":"text"}`)}
	journal.messages = append(journal.messages,
		modelapi.Message{Role: "assistant", Calls: []toolapi.Call{read}},
		modelapi.Message{Role: "tool", ToolCallID: read.ID, ToolName: read.Name, Content: observed, TransientObservation: true},
	)
	path := filepath.Join(root, "note.txt")
	if err := os.WriteFile(path, []byte("status: pending\n"), 0600); err != nil {
		t.Fatal(err)
	}
	args := map[string]any{"path": "note.txt", "edits": []map[string]string{{"old_text": "status: pending", "new_text": "status: done"}}}
	missing := nativeCall("edit_file", args)
	args["reason"] = "Record completed status"
	g := &generator{replies: []string{missing, nativeCall("edit_file", args), "Status updated."}}
	r.Generator = g
	result, err := r.Run(t.Context(), "Update the status.", approve, emit)
	if err != nil || result.Status != runrecord.Answered || len(g.requests) != 3 {
		t.Fatalf("result=%+v requests=%d err=%v", result, len(g.requests), err)
	}
	if !containsPage(g.requests[1].Messages, observed) || containsPage(g.requests[2].Messages, observed) {
		t.Fatal("inspection evidence should remain through correction and retire after the corrected call")
	}
	for _, message := range g.requests[1].Messages {
		if message.Role == "tool" && message.ToolCallID == "call_1" {
			if !strings.Contains(message.Content, `"field":"reason"`) || !strings.Contains(message.Content, "missing reason argument") || strings.Contains(message.Content, "old_text must match") {
				t.Fatalf("unexpected reason correction: %s", message.Content)
			}
			if got, err := os.ReadFile(path); err != nil || string(got) != "status: done\n" {
				t.Fatalf("corrected edit produced %q: %v", got, err)
			}
			return
		}
	}
	t.Fatal("missing paired tool result")
}

func TestRepeatedMissingReasonStopsBeforeEffects(t *testing.T) {
	r, _, root, journal, _ := realBatchRunner(t, nil)
	invalid := nativeCall("write_file", map[string]any{"path": "unexpected", "content": "x"})
	g := &generator{replies: []string{invalid, invalid, invalid, "script continued past correction budget"}}
	r.Generator = g
	result, err := r.Run(t.Context(), "Create a file.", approve, emit)
	if err == nil || !strings.Contains(err.Error(), "after 2 corrections") || len(g.requests) != 3 || result.Status != runrecord.Stopped {
		t.Fatalf("result=%+v requests=%d err=%v", result, len(g.requests), err)
	}
	if _, err := os.Stat(filepath.Join(root, "unexpected")); !os.IsNotExist(err) {
		t.Fatalf("rejected write had an effect: %v", err)
	}
	count := 0
	for _, message := range journal.messages {
		if message.Role == "assistant" && message.PreserveObservations {
			count++
		}
	}
	if count != 3 {
		t.Fatalf("rejected decisions=%d", count)
	}
}
