// Purpose: Verify output folding works with legacy and enhanced terminal key events.
// Pattern: Virtual terminal regression tests.
// Usage: go test ./internal/tui/session -run TestOutputToggle.
// Related: internal/tui/session/keys.go, internal/tui/transcript/events.go, internal/tui/viewport/viewport.go.
package session

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v3"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/tui/transcript"
)

func TestOutputToggleTerminalProtocols(t *testing.T) {
	for _, enhanced := range []bool{false, true} {
		for _, restored := range []bool{false, true} {
			a := fixture(t)
			a.editor.Set("draft")
			a.worker = workerAgent
			body, err := json.Marshal(map[string]any{"Stdout": strings.Repeat("detail row\n", 40), "ExitCode": 0, "Status": "exited"})
			if err != nil {
				t.Fatal(err)
			}
			event := runrecord.Event{Kind: "tool_succeeded", Text: "Process exited; exit code 0", Call: &toolapi.Call{Name: "execute_command", Arguments: []byte(`{"command":"cat large.txt"}`)}, Output: body}
			if restored {
				a.output.Prepend((transcript.Formatter{Redact: a.opts.Vault.Redact}).Event(event))
			} else {
				a.receiveRuntime(&event)
			}
			a.render()
			key := tcell.NewEventKey(tcell.KeyCtrlO, "", 0)
			if enhanced {
				key = tcell.NewEventKeyEx(tcell.KeyRune, "o", tcell.ModLCtrl, true, tcell.KeyO, 1)
			}
			a.keyEvent(key)
			a.render()
			visible := strings.Join(screenRows(a, 22), "\n")
			if !a.output.Collapsed || strings.Contains(visible, "detail row") || a.editor.String() != "draft" {
				t.Fatalf("enhanced=%t restored=%t collapsed=%t draft=%q: %s", enhanced, restored, a.output.Collapsed, a.editor.String(), visible)
			}
			if keymapLabels(t, a)["^O"] != "도구 펴기" {
				t.Fatal("output advertised the wrong key action")
			}
			a.keyEvent(key)
			a.render()
			if a.output.Collapsed || !strings.Contains(strings.Join(screenRows(a, 22), "\n"), "detail row") {
				t.Fatal("output did not expand on the second keypress")
			}
			if keymapLabels(t, a)["^O"] != "도구 접기" {
				t.Fatal("output advertised the wrong key action")
			}
		}
	}
}

func TestOutputToggleDoesNotInterpretPastedText(t *testing.T) {
	a := fixture(t)
	a.paste = true
	a.keyEvent(tcell.NewEventKeyEx(tcell.KeyRune, "o", tcell.ModCtrl, true, tcell.KeyO, 1))
	if a.output.Collapsed || a.stopNotice != "" {
		t.Fatal("pasted content executed an output shortcut")
	}
}

func screenRows(a *app, count int) []string {
	width, _ := a.screen.Size()
	rows := make([]string, count)
	for y := range rows {
		rows[y] = screenLine(a.screen, y, width)
	}
	return rows
}

func TestOutputToggleHintRequiresOutput(t *testing.T) {
	a := fixture(t)
	if keymapLabels(t, a)["^O"] != "" {
		t.Fatal("fold hint advertised without any foldable output")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyCtrlO, "", 0))
	if a.output.Collapsed {
		t.Fatal("empty transcript changed the default for future outputs")
	}
}
