// Purpose: Verify spaced tool transcripts show actual output once and redact literal values.
// Pattern: Terminal projection tests.
// Usage: go test ./internal/tui/session -run 'TestToolRecord|TestToolFailure'.
// Related: internal/tui/transcript/tools.go, internal/tui/session/events.go, internal/tui/session/history.go.
package session

import (
	"encoding/json"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/tui/transcript"
	"strings"
	"testing"
)

func TestToolRecordLiveMatchesRestored(t *testing.T) {
	a := fixture(t)
	call := toolapi.Call{ID: "call", Name: "execute_command", Arguments: json.RawMessage(`{"command":"cat README.md"}`)}
	event := runrecord.Event{Kind: "tool_started", CallID: call.ID, Call: &call}
	a.receive(notice{Event: &event})
	if len(a.output.Blocks) != 0 || !strings.Contains(activityTranscript(a), "명령 실행") {
		t.Fatal("start should only show pending call")
	}
	event.Kind, event.Text, event.Output = "tool_succeeded", "9 bytes read", json.RawMessage(`{"Stdout":"one\ntwo","ExitCode":0,"Status":"exited"}`)
	a.receive(notice{Event: &event})
	if a.status != "" || len(a.output.Blocks) != 1 {
		t.Fatal("pending call was not replaced by one completion record")
	}
	want := "\n  명령 실행\n\n    명령: cat README.md\n\n    결과\n    표준 출력\n    one\n    two\n    실행 성공\n"
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
	if got := screenLine(a.screen, 2, 80); !strings.HasPrefix(got, "  명령 실행") {
		t.Fatalf("name not indented: %q", got)
	}
	if got := screenLine(a.screen, 4, 80); !strings.HasPrefix(got, "    명령:") {
		t.Fatalf("parameters not indented: %q", got)
	}
}

func TestToolFailureAndDenialHaveDetails(t *testing.T) {
	a := fixture(t)
	a.opts.Vault.Remember(`secret"key`)
	for _, kind := range []string{"tool_failed", "denied", "tool_interrupted"} {
		event := runrecord.Event{Kind: kind, CallID: "call", Text: "operation stopped", Call: &toolapi.Call{ID: "call", Name: "execute_command", Arguments: json.RawMessage(`{"command":"echo secret\"key"}`)}}
		text := (transcript.Formatter{Redact: a.opts.Vault.Redact}).Tool(event)
		if !strings.Contains(text, "결과\n    operation stopped") || strings.Contains(text, `secret"key`) || !strings.Contains(text, "[redacted]") {
			t.Fatalf("unsafe/missing failure record: %q", text)
		}
	}
	a.opts.Vault.Remember("provider-secret")
	event := runrecord.Event{Kind: "tool_failed", Text: "exit 1", Call: &toolapi.Call{Name: "execute_command", Arguments: json.RawMessage(`{}`)}, Output: json.RawMessage(`{"Stdout":"provider-secret\nkeep","Stderr":"error","ExitCode":1,"Status":"exited"}`)}
	text := (transcript.Formatter{Redact: a.opts.Vault.Redact}).Tool(event)
	if strings.Contains(text, "provider-secret") || !strings.Contains(text, "[redacted]") || !strings.Contains(text, "표준 오류: error") || !strings.Contains(text, "실행 실패") || strings.Contains(text, "exit 1") {
		t.Fatalf("output/failure missing or unsafe: %q", text)
	}
}

func TestToolReadLineNumbersAlignOnScreen(t *testing.T) {
	a := fixture(t)
	a.receive(notice{Event: &runrecord.Event{Kind: "tool_succeeded", Call: &toolapi.Call{Name: "execute_command", Arguments: []byte(`{"command":"nl a"}`)}, Output: []byte(`{"Stdout":"  1 | first\n 10 | tenth\n100 | hundredth\n","ExitCode":0,"Status":"exited"}`)}})
	for _, width := range []int{80, 32} {
		a.screen.SetSize(width, 24)
		a.render()
		column, found := -1, 0
		for row := 0; row < 22; row++ {
			line := screenLine(a.screen, row, width)
			if at := strings.Index(line, " | "); at >= 0 {
				found++
				if column >= 0 && column != at {
					t.Fatalf("misaligned row: %q", line)
				}
				column = at
			}
		}
		if found != 3 {
			t.Fatalf("width %d: expected 3 numbered rows, got %d", width, found)
		}
	}
}
