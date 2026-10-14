// Purpose: Verify live and restored review display reveals only outcome, never internal work or ballot reasons.
// Pattern: Transcript projection tests.
// Usage: go test ./internal/tui/transcript -run TestConclave.
// Related: internal/tui/transcript/conclave.go, internal/tui/transcript/tools.go, internal/tui/transcript/events.go.
package transcript

import (
	"github.com/keonho-kim/v-code/internal/agent/conclave"
	"github.com/keonho-kim/v-code/internal/agent/run"
	"github.com/keonho-kim/v-code/internal/agent/tool"
	"strings"
	"testing"
)

func TestConclaveDisplayHidesInternalRecords(t *testing.T) {
	f := Formatter{Redact: func(s string) string { return s }}
	event := run.Event{Kind: "conclave", Call: &tool.Call{ID: "c", Name: "edit_file", Arguments: []byte(`{"path":"a"}`)}, Conclave: &conclave.Result{Votes: []conclave.Vote{{Reason: "private reviewer detail"}}}}
	blocks := f.Event(event)
	if len(blocks) != 1 || !strings.Contains(blocks[0].Text, "edit_file · Conclave Fail") || !strings.Contains(blocks[0].Text, ReviewRejectedMessage) || strings.Contains(blocks[0].Text, "private") {
		t.Fatalf("bad transcript: %+v", blocks)
	}
	event.Conclave.Passed = true
	if len(f.Event(event)) != 0 {
		t.Fatal("pass added an internal ballot record")
	}
	event.Kind = "tool_succeeded"
	event.Text = "completed"
	if text := f.Tool(event); !strings.Contains(text, "edit_file · Conclave Pass") || strings.Contains(text, "private") {
		t.Fatal(text)
	}
	for _, kind := range []string{"goal_state", "monitor", "tool_proposed"} {
		if len(f.Event(run.Event{Kind: kind, Text: "internal work"})) != 0 {
			t.Fatalf("%s leaked", kind)
		}
	}
}
