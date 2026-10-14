// Purpose: Verify spaced tool transcripts show actual output once and redact literal values.
// Pattern: Terminal projection tests.
// Usage: go test ./internal/tui/session -run 'TestToolRecord|TestToolFailure'.
// Related: internal/tui/transcript/tools.go, internal/tui/session/events.go, internal/tui/session/history.go.
package session

import (
	"encoding/json"
	"github.com/keonho-kim/v-code/internal/agent/run"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/tui/transcript"
	"strings"
	"testing"
)

func TestToolRecordLiveMatchesRestored(t *testing.T) {
	a := fixture(t)
	call := toolapi.Call{ID: "call", Name: "read_file", Arguments: json.RawMessage(`{"path":"README.md"}`)}
	event := run.Event{Kind: "tool_started", CallID: call.ID, Call: &call}
	a.receive(notice{Event: &event})
	if len(a.output.Blocks) != 0 || !strings.Contains(a.status, "read_file") {
		t.Fatal("start should only show pending call")
	}
	event.Kind, event.Text, event.Output = "tool_succeeded", "9 bytes read", json.RawMessage(`{"content":"one\ntwo","path":"README.md"}`)
	a.receive(notice{Event: &event})
	if a.status != "" || len(a.output.Blocks) != 1 {
		t.Fatal("pending call was not replaced by one completion record")
	}
	want := "\n  read_file\n\n    path: README.md\n\n    Output\n    content: one\n    two\n    path: README.md\n"
	if got := a.output.Blocks[0].Text; got != want {
		t.Fatalf("tool record=%q; want %q", got, want)
	}
	restored := (transcript.Formatter{Redact: a.opts.Vault.Redact}).Event(event)
	if len(restored) != 1 || restored[0].Text != want {
		t.Fatalf("restored record: %+v", restored)
	}
	a.receive(notice{Done: true})
	if len(a.output.Blocks) != 1 {
		t.Fatal("completion state added transcript boilerplate")
	}
	a.render()
	if got := screenLine(a.screen, 2, 80); !strings.HasPrefix(got, "  read_file") {
		t.Fatalf("name not indented: %q", got)
	}
	if got := screenLine(a.screen, 4, 80); !strings.HasPrefix(got, "    path:") {
		t.Fatalf("parameters not indented: %q", got)
	}
}

func TestToolFailureAndDenialHaveDetails(t *testing.T) {
	a := fixture(t)
	a.opts.Vault.Remember(`secret"key`)
	for _, kind := range []string{"tool_failed", "denied", "tool_interrupted"} {
		event := run.Event{Kind: kind, CallID: "call", Text: "operation stopped", Call: &toolapi.Call{ID: "call", Name: "execute_command", Arguments: json.RawMessage(`{"command":"echo secret\"key"}`)}}
		text := (transcript.Formatter{Redact: a.opts.Vault.Redact}).Tool(event)
		if !strings.Contains(text, "Output\n    operation stopped") || strings.Contains(text, `secret"key`) || !strings.Contains(text, "[redacted]") {
			t.Fatalf("unsafe/missing failure record: %q", text)
		}
	}
	a.opts.Vault.Remember("provider-secret")
	event := run.Event{Kind: "tool_failed", Text: "exit 1", Call: &toolapi.Call{Name: "execute_command", Arguments: json.RawMessage(`{}`)}, Output: json.RawMessage(`{"stdout":"provider-secret\nkeep","stderr":"error"}`)}
	text := (transcript.Formatter{Redact: a.opts.Vault.Redact}).Tool(event)
	if strings.Contains(text, "provider-secret") || !strings.Contains(text, "[redacted]") || !strings.Contains(text, "stderr: error") || !strings.Contains(text, "Failed: exit 1") {
		t.Fatalf("output/failure missing or unsafe: %q", text)
	}
}
