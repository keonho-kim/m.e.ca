// Purpose: Verify every completed tool result starts expanded and folds without altering history.
// Pattern: Virtual terminal regression tests.
// Usage: go test ./internal/tui/session -run TestAllToolResults.
// Related: internal/tui/transcript/events.go, internal/tui/viewport/viewport.go, internal/tui/session/keys.go.
package session

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v3"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func TestAllToolResultsStartExpandedAndFold(t *testing.T) {
	for _, test := range []struct{ name, arguments, output, detail string }{
		{"execute_command", `{"command":"printf detail"}`, `{"Stdout":"command-detail","ExitCode":0,"Status":"exited"}`, "command-detail"},
		{"write_file", `{"path":"hello.txt","content":"file-detail"}`, `{"changed":true}`, "file-detail"},
		{"edit_file", `{"path":"hello.txt","edits":[{"old_text":"old-detail","new_text":"edit-detail"}]}`, `{"changed":true}`, "edit-detail"},
		{"create_directory", `{"paths":["src","tests"]}`, `{"changed":true,"created":["src","tests"]}`, "폴더 생성 완료"},
	} {
		t.Run(test.name, func(t *testing.T) {
			a := fixture(t)
			event := runrecord.Event{Kind: "tool_succeeded", Text: "done", Call: &toolapi.Call{Name: test.name, Arguments: json.RawMessage(test.arguments)}, Output: json.RawMessage(test.output)}
			if test.name == "write_file" || test.name == "edit_file" {
				event.ChangePreview = "--- old\n+++ hello.txt\n@@ -1 +1 @@\n-old\n+" + test.detail + "\n"
			}
			a.receiveRuntime(&event)
			original := a.output.Blocks[0].Text
			if !strings.Contains(original, test.detail) {
				t.Fatalf("default output was folded or missing: %q", original)
			}
			a.keyEvent(tcell.NewEventKey(tcell.KeyCtrlO, "", 0))
			a.render()
			folded := a.output.Blocks[0].Text
			if original == folded || !strings.Contains(folded, a.language.Text(test.name)) || !strings.Contains(folded, "접힘") {
				t.Fatalf("tool was not folded: %q", folded)
			}
			if !strings.Contains(strings.Join(screenRows(a, 24-a.footerRows), "\n"), "접힘") || !strings.Contains(footerKeyText(a), "^O") {
				t.Fatal("fold state or Ctrl+O shortcut was not visible")
			}
			a.keyEvent(tcell.NewEventKey(tcell.KeyCtrlO, "", 0))
			if a.output.Blocks[0].Text != original {
				t.Fatal("expansion changed the original displayed result")
			}
			if string(event.Output) != test.output || string(event.Call.Arguments) != test.arguments {
				t.Fatal("display folding modified execution facts")
			}
		})
	}
}
