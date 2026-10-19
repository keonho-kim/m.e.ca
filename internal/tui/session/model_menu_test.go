// Purpose: Verify connection-field clearing and inline Thinking selection preserve explicit user choices.
// Pattern: Terminal behavior tests.
// Usage: go test ./internal/tui/session -run 'TestEndpoint|TestInlineThinking'.
// Related: internal/tui/session/menus.go, internal/tui/session/menu_keys.go, internal/tui/session/render.go.
package session

import (
	"github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/vt"
	"strings"
	"testing"
)

func TestEndpointClearRemovesSuffixAndRejectsEmpty(t *testing.T) {
	a := fixture(t)
	a.page = "endpoint"
	a.connectionKind = 4
	a.menuInput.Set("http://localhost:8000/v1")
	for i := 0; i < 3; i++ {
		a.keyEvent(tcell.NewEventKey(tcell.KeyLeft, "", 0))
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyCtrlU, "", 0))
	a.render()
	if a.menuInput.String() != "" || strings.Contains(screenLine(a.screen, 2, 80), "v1") {
		t.Fatal("clearing retained an invisible endpoint suffix")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	if a.page != "endpoint" || a.status == "" {
		t.Fatal("empty endpoint silently restored the default address")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyRune, "http://127.0.0.1:9000", 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	if a.page != "key" || a.endpoint != "http://127.0.0.1:9000" {
		t.Fatalf("entered endpoint changed: %q", a.endpoint)
	}
}

func TestEndpointCursorTracksEditingAndHorizontalScroll(t *testing.T) {
	a := fixture(t)
	a.page = "endpoint"
	a.menuInput.Set("http://localhost:8000/v1")
	a.keyEvent(tcell.NewEventKey(tcell.KeyLeft, "", 0))
	a.render()
	tty, ok := a.screen.Tty()
	if !ok {
		t.Fatal("test terminal unavailable")
	}
	terminal, ok := tty.(vt.MockTerm)
	if !ok {
		t.Fatal("test terminal type unavailable")
	}
	if err := terminal.Drain(); err != nil {
		t.Fatal(err)
	}
	if int(terminal.Pos().X) != 2+a.menuInput.Cursor {
		t.Fatal("visible cursor differs from editor cursor")
	}
	a.screen.SetSize(20, 10)
	a.menuInput.End()
	a.render()
	if err := terminal.Drain(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(screenLine(a.screen, 2, 20), "8000/v1") || terminal.Pos().X != 19 {
		t.Fatal("long endpoint tail or cursor is hidden")
	}
}

func TestInlineThinkingShowsCurrentAndChoosesWithArrows(t *testing.T) {
	a := fixture(t)
	off := false
	a.selection.Thinking, a.selection.ThinkingSupported = &off, true
	a.openModel()
	a.render()
	if line := screenLine(a.screen, 3, 80); !strings.Contains(line, "추론 켜짐 / 꺼짐") {
		t.Fatalf("inline control missing: %q", line)
	}
	_, onStyle, _ := a.screen.Get(5, 3)
	_, offStyle, _ := a.screen.Get(12, 3)
	if !offStyle.HasBold() || !onStyle.HasDim() {
		t.Fatal("current OFF is not distinguished from ON")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyDown, "", 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyLeft, "", 0))
	a.render()
	_, onStyle, _ = a.screen.Get(5, 3)
	_, offStyle, _ = a.screen.Get(12, 3)
	if !onStyle.HasBold() || !offStyle.HasDim() {
		t.Fatal("arrow selection did not emphasize ON")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	if a.page != "global" || a.changeValue != "ON" || *a.selection.Thinking {
		t.Fatal("Thinking changed before choosing the settings destination")
	}
}
