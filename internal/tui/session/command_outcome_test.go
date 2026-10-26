// Purpose: Verify bilingual command outcomes stay readable during live output, restoration and folding.
// Pattern: Virtual terminal regression tests.
// Usage: go test ./internal/tui/session -run TestCommandOutcomeScreen.
// Related: internal/tui/transcript/process_result.go, internal/tui/session/runtime_events.go, internal/tui/session/keys.go.
package session

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v3"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/tui/locale"
	"github.com/keonho-kim/meca/internal/tui/transcript"
)

func TestCommandOutcomeScreen(t *testing.T) {
	for _, lang := range []locale.Language{locale.Korean, locale.English} {
		for _, restored := range []bool{false, true} {
			a := fixture(t)
			a.language = lang
			event := runrecord.Event{
				Kind: "tool_failed", Text: "process_failed: Process exited; exit code 1; next: internal guidance",
				Call:      &tool.Call{Name: "execute_command", Arguments: []byte(`{"command":"check"}`)},
				Output:    []byte(`{"Stderr":"missing package","ExitCode":1,"Status":"exited"}`),
				Operation: &tool.Operation{Failure: &tool.Failure{Code: tool.ProcessFailed}},
			}
			if restored {
				a.output.Prepend((transcript.Formatter{Language: lang, Redact: a.opts.Vault.Redact}).Event(event))
			} else {
				a.receiveRuntime(&event)
			}
			for _, toggle := range []bool{false, true, true} {
				if toggle {
					a.keyEvent(tcell.NewEventKey(tcell.KeyCtrlO, "", 0))
				}
				a.render()
				shown := strings.Join(screenRows(a, 22), "\n")
				if !strings.Contains(shown, lang.Text("Command failed")) || !strings.Contains(shown, "missing package") {
					t.Fatalf("language=%v restored=%v collapsed=%v: outcome or diagnostic missing: %s", lang, restored, a.output.Collapsed, shown)
				}
				for _, hidden := range []string{"exit:", "exit code", "종료 코드", "process_failed", "internal guidance"} {
					if strings.Contains(shown, hidden) {
						t.Fatalf("command metadata %q leaked: %s", hidden, shown)
					}
				}
			}
		}
	}
}
