// Purpose: Verify localized command outcomes replace exit-code metadata without changing execution records.
// Pattern: Presentation contract regression tests.
// Usage: go test ./internal/tui/transcript -run TestCommandOutcome.
// Related: internal/tui/transcript/process_result.go, internal/tui/transcript/tool_summary.go, internal/tui/locale/tools.go.
package transcript

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/keonho-kim/v-code/internal/agent/run"
	"github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/tui/locale"
)

func TestCommandOutcomeLocalized(t *testing.T) {
	for _, tc := range []struct {
		name, kind, body, failure, korean, english string
	}{
		{"success", "tool_succeeded", `{"Stdout":"result","ExitCode":0,"Status":"exited"}`, "", "실행 성공", "Command succeeded"},
		{"failure", "tool_failed", `{"Stderr":"check failed","ExitCode":1,"Status":"exited"}`, tool.ProcessFailed, "실행 실패", "Command failed"},
		{"quiet failure", "tool_failed", `{"ExitCode":2,"Status":"exited"}`, tool.ProcessFailed, "실행 실패", "Command failed"},
		{"interrupted", "tool_failed", `{"Stdout":"partial","ExitCode":-1,"Status":"interrupted"}`, tool.Interrupted, "실행 중단", "Command interrupted"},
		{"timeout", "tool_failed", `{"Stdout":"partial","ExitCode":-1,"Status":"timeout"}`, tool.Timeout, "시간 초과", "Command timed out"},
		{"start failure", "tool_failed", `{"Started":false,"ExitCode":-1,"Status":"execution_failed"}`, tool.ExecutionFailed, "실행 실패", "Command failed"},
		{"capture failure", "tool_failed", `{"Stdout":"completed","ExitCode":0,"Status":"exited"}`, tool.ExecutionFailed, "실행 실패", "Command failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, lang := range []locale.Language{locale.Korean, locale.English} {
				raw := json.RawMessage(tc.body)
				original := bytes.Clone(raw)
				event := run.Event{Kind: tc.kind, Text: "Process exited; exit code 1", Call: &tool.Call{Name: "execute_command", Arguments: []byte(`{"command":"check"}`)}, Output: raw}
				if tc.failure != "" {
					event.Operation = &tool.Operation{Failure: &tool.Failure{Code: tc.failure, Detail: "diagnostic secret", Next: "internal retry instruction"}}
				}
				f := Formatter{Language: lang, Redact: func(s string) string { return strings.ReplaceAll(s, "secret", "[redacted]") }}
				want := tc.korean
				if lang == locale.English {
					want = tc.english
				}
				block := f.Event(event)[0]
				if tc.name == "failure" && !strings.Contains(block.Summary, "check failed") {
					t.Fatal("collapsed command failure lost its diagnostic:", block.Summary)
				}
				for _, shown := range []string{block.Text, block.Summary} {
					if strings.Count(shown, want) != 1 {
						t.Fatalf("outcome %q should appear once: %s", want, shown)
					}
					for _, hidden := range []string{"exit:", "exit code", "종료 코드", "ExitCode", "internal retry instruction", "diagnostic secret"} {
						if strings.Contains(shown, hidden) {
							t.Fatalf("internal metadata %q leaked: %s", hidden, shown)
						}
					}
					if tc.failure == tool.ExecutionFailed && !strings.Contains(shown, "diagnostic [redacted]") {
						t.Fatalf("execution diagnostic lost: %s", shown)
					}
				}
				if !bytes.Equal(raw, original) || f.Tool(event) != block.Text {
					t.Fatal("presentation changed the record or differs on restoration")
				}
			}
		})
	}
}

func TestCommandOutcomePreservesLiteralExitText(t *testing.T) {
	event := run.Event{Kind: "tool_succeeded", Call: &tool.Call{Name: "execute_command", Arguments: []byte(`{"command":"printf output"}`)}, Output: []byte(`{"Stdout":"exit: 1\nexit: 0","ExitCode":0,"Status":"exited"}`)}
	f := Formatter{Redact: func(s string) string { return s }}
	shown := f.Tool(event)
	if !strings.Contains(shown, "exit: 1\n    exit: 0") || !strings.Contains(shown, "실행 성공") {
		t.Fatalf("literal command output was changed: %s", shown)
	}
}

func TestCommandOutcomeInvalidResultDoesNotClaimSuccess(t *testing.T) {
	f := Formatter{Redact: func(s string) string { return s }}
	for _, body := range []string{`{"ExitCode":null,"Status":"exited"}`, `{"ExitCode":"0","Status":"exited"}`, `{"ExitCode":0,"Status":42}`} {
		event := run.Event{Kind: "tool_failed", Text: "unrecognized command result", Call: &tool.Call{Name: "execute_command"}, Output: []byte(body)}
		block := f.Event(event)[0]
		for _, text := range []string{block.Text, block.Summary} {
			if !strings.Contains(text, event.Text) || strings.Contains(text, "실행 성공") {
				t.Fatal("invalid command result was presented as success:", text)
			}
		}
	}
}
