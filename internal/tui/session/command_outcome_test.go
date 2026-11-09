// Purpose: Verify bilingual command outcomes stay readable during live output, restoration and folding.
// Pattern: Virtual terminal regression tests.
// Usage: go test ./internal/tui/session -run TestCommandOutcomeScreen.
// Related: internal/tui/transcript/process_result.go, internal/tui/session/runtime_events.go, internal/tui/session/keys.go.
package session

import (
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"

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
			a := screenFixture(t)
			a.language = lang
			event := runrecord.Event{
				Kind: "tool_failed", Text: "process_failed: Process exited; exit code 1; next: internal guidance",
				Call:      &tool.Call{Name: "execute_command", Arguments: []byte(`{"command":"check"}`)},
				Output:    []byte(`{"Stderr":"missing package","ExitCode":1,"Status":"exited"}`),
				Operation: &tool.Operation{Process: &tool.ProcessOutcome{CommandID: "cmd_failed", Status: "exited", ExitCode: 1, Diagnostic: "missing package"}, Failure: &faults.Failure{Code: diagcatalog.ProcessFailed}},
			}
			if restored {
				a.chat.output.Prepend((transcript.Formatter{Language: lang, Redact: a.opts.Vault.Redact}).Event(event))
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
					t.Fatalf("language=%v restored=%v collapsed=%v: outcome or diagnostic missing: %s", lang, restored, a.chat.output.Collapsed, shown)
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

func TestCommandFailureCauseSurvivesLazyHistory(t *testing.T) {
	a := fixture(t)
	event := runrecord.Event{Kind: "tool_failed", CallID: "failed", Text: "Process exited; exit code 1",
		Call:   &tool.Call{ID: "failed", Name: "execute_command", Arguments: []byte(`{"command":"check"}`)},
		Output: []byte(`{"Stdout":"original output","Stderr":"missing package","ExitCode":1,"Status":"exited"}`),
		Operation: &tool.Operation{
			Process: &tool.ProcessOutcome{CommandID: "cmd_failed", Status: "exited", ExitCode: 1, Diagnostic: "missing package"},
			Failure: &faults.Failure{Code: diagcatalog.ProcessFailed},
		},
	}
	if err := a.opts.Store.RecordEvent(a.ctx, a.opts.Session.ID, event); err != nil {
		t.Fatal(err)
	}
	if err := a.loadHistory(); err != nil {
		t.Fatal(err)
	}
	a.render()
	if shown := activityTranscript(a); !strings.Contains(shown, "missing package") || strings.Contains(shown, "original output") || a.chat.details.cancel != nil {
		t.Fatalf("collapsed history lost its cause or loaded full output: %q", shown)
	}
	expandVisibleDetails(t, a)
	if shown := activityTranscript(a); !strings.Contains(shown, "missing package") || !strings.Contains(shown, "original output") {
		t.Fatalf("expanded history lost original command output: %q", shown)
	}
}
