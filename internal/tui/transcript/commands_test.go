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
	event := runrecord.Event{Kind: "tool_succeeded", Call: &tool.Call{Name: "wait_command", Arguments: json.RawMessage("{\"command_id\":\"cmd_owned\",\"wait_seconds\":0}")}, Output: json.RawMessage("{\"command_id\":\"cmd_owned\",\"Status\":\"running\",\"Stdout\":\"still working\"}")}
	block := f.Event(event)[0]
	if !strings.Contains(block.Text, "명령 실행 중") || !strings.Contains(block.Text, "still working") || strings.Contains(block.Text, "실행 성공") {
		t.Fatal(block.Text)
	}
	list := f.commandList([]byte("{\"runs\":[{\"command_id\":\"cmd_owned\",\"command\":\"test\",\"status\":\"running\",\"elapsed_millis\":180000}]}"))
	if strings.Contains(list.text, "\"runs\"") || !strings.Contains(list.text, "cmd_owned") || !strings.Contains(list.text, "00 : 03 : 00") {
		t.Fatal(list.text)
	}
	live := f.CommandActivity([]tool.CommandRun{{ID: "cmd_owned", Command: "test", Status: "running", ElapsedMillis: 180000, LastOutputMillis: 20000}})
	if !strings.Contains(live, "마지막 출력") || !strings.Contains(live, "00 : 00 : 20") {
		t.Fatal(live)
	}
}
