// Purpose: Verify review findings, locations and errors are visible and redacted in live and restored output.
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

func TestConclaveDisplayShowsRejectionReasons(t *testing.T) {
	f := Formatter{Redact: func(s string) string { return s }}
	event := run.Event{Kind: "conclave", Call: &tool.Call{ID: "c", Name: "edit_file", Arguments: []byte(`{"path":"a"}`)}, Conclave: &conclave.Result{Verdict: conclave.Verdict{Decision: "reject", Findings: []conclave.Finding{{Path: "a", Side: "request", Reason: "private reviewer detail"}}}}}
	blocks := f.Event(event)
	if len(blocks) != 1 || !strings.Contains(blocks[0].Text, "edit_file · Conclave Fail") || !strings.Contains(blocks[0].Text, ReviewRejectedMessage) || !strings.Contains(blocks[0].Text, "private reviewer detail") {
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
	for _, kind := range []string{"monitor", "tool_proposed"} {
		if len(f.Event(run.Event{Kind: kind, Text: "internal work"})) != 0 {
			t.Fatalf("%s leaked", kind)
		}
	}
}

func TestConclaveFindingsShowLocationsAndRedact(t *testing.T) {
	f := Formatter{Redact: func(s string) string { return strings.ReplaceAll(s, "secret", "[REDACTED]") }}
	event := run.Event{Kind: "conclave", Call: &tool.Call{Name: "edit_file"}, Conclave: &conclave.Result{Verdict: conclave.Verdict{Decision: "reject", Findings: []conclave.Finding{
		{Path: "secret.go", Side: "after", Line: 12, Reason: "first reason secret\nadditional detail"},
		{Path: "b.go", Side: "before", Line: 7, Reason: "second reason"},
		{Path: "c.go", Side: "request", Reason: "request mismatch"},
	}}}}
	shown := f.Conclave(event)
	for _, want := range []string{"[REDACTED].go:12 (변경 후)", "b.go:7 (변경 전)", "c.go (요청)", "first reason [REDACTED]", "second reason", "request mismatch", "\n    additional detail"} {
		if !strings.Contains(shown, want) {
			t.Fatalf("missing %q: %s", want, shown)
		}
	}
	if strings.Contains(shown, "secret") {
		t.Fatal("secret exposed")
	}
	blocks := f.Event(event)
	if len(blocks) != 1 || blocks[0].Text != shown {
		t.Fatal("restored display differs")
	}
	event.Conclave = &conclave.Result{Error: "review timed out secret"}
	if shown = f.Conclave(event); !strings.Contains(shown, "심사 오류: review timed out [REDACTED]") {
		t.Fatal(shown)
	}
}
