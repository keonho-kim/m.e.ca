// Purpose: Verify running commands and owned command lists render compact localized results.
// Pattern: Presentation contract tests.
// Usage: go test ./internal/tui/transcript -run TestOwnedCommands.
// Related: internal/tui/transcript/commands.go, internal/tui/transcript/process_result.go.
package transcript

import (
	"encoding/json"
	"strings"
	"testing"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/tui/locale"
)

func TestOwnedCommandsPresentation(t *testing.T) {
	f := Formatter{Language: locale.Korean, Redact: func(text string) string { return text }}
	event := runrecord.Event{Kind: "tool_succeeded", Call: &tool.Call{Name: "wait_command", Arguments: json.RawMessage(`{"command_id":"cmd_owned","wait_seconds":0}`)}, Output: json.RawMessage(`{"command_id":"cmd_owned","Status":"running","Stdout":"still working","next":"Use wait_command, cancel_command or list_command_runs"}`)}
	block := f.Event(event)[0]
	if !strings.Contains(block.Text, "명령 실행 중") || !strings.Contains(block.Text, "still working") || strings.Contains(block.Text, "실행 성공") {
		t.Fatal(block.Text)
	}
	if strings.Contains(block.Text, "list_command_runs") || strings.Contains(block.Text, "Use wait_command") {
		t.Fatalf("model management guidance leaked into terminal output: %s", block.Text)
	}
	list := f.commandList([]byte("{\"runs\":[{\"command_id\":\"cmd_owned\",\"command\":\"test\",\"status\":\"running\",\"elapsed_millis\":180000}]}"))
	if strings.Contains(list.text, "\"runs\"") || !strings.Contains(list.text, "cmd_owned") || !strings.Contains(list.text, "00 : 03 : 00") {
		t.Fatal(list.text)
	}
	age := int64(20000)
	live := f.CommandActivity([]tool.CommandRun{{ID: "cmd_owned", Command: "test", Status: "running", ElapsedMillis: 180000, LastOutputMillis: &age}})
	if !strings.Contains(live, "마지막 출력") || !strings.Contains(live, "00 : 00 : 20") {
		t.Fatal(live)
	}
}

func TestOwnedCommandActivityPreservesFullCommand(t *testing.T) {
	f := Formatter{Language: locale.Korean, Redact: func(text string) string { return strings.ReplaceAll(text, "private-value", "[redacted]") }}
	command := "printf 'private-value " + strings.Repeat("content-", 30) + "' &&\ncat docs/final-section.md"
	shown := f.CommandActivity([]tool.CommandRun{{ID: "cmd_owned", Command: command, Status: "running"}})
	if !strings.Contains(shown, "cat docs/final-section.md") || !strings.Contains(shown, strings.Repeat("content-", 30)) || !strings.Contains(shown, "\n    cat") {
		t.Fatalf("live command was shortened or lost line breaks: %s", shown)
	}
	if strings.Contains(shown, "private-value") || !strings.Contains(shown, "[redacted]") {
		t.Fatalf("full live command lost redaction: %s", shown)
	}
}

func TestOwnedCommandActivityWithoutOutput(t *testing.T) {
	for _, lang := range []locale.Language{locale.Korean, locale.English} {
		f := Formatter{Language: lang, Redact: func(text string) string { return text }}
		shown := f.CommandActivity([]tool.CommandRun{{Command: "quiet-command", Status: "running", ElapsedMillis: 120000}})
		if !strings.Contains(shown, lang.Text("No output yet")) || strings.Contains(shown, lang.Format("Last output %s ago", commandDuration(120000))) {
			t.Fatalf("silent command invented a last-output time: %s", shown)
		}
		if !strings.Contains(shown, lang.Format("Elapsed %s", commandDuration(120000))) {
			t.Fatalf("silent command lost elapsed time: %s", shown)
		}
	}
}
