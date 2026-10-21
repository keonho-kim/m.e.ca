// Purpose: Verify drag-copy handles visible chat/input and takes precedence over app exit.
// Pattern: Virtual terminal integration tests.
// Usage: go test ./internal/tui/session -run TestTextSelection.
// Related: internal/tui/session/text_selection.go, internal/tui/selection/selection.go.
package session

import (
	"github.com/gdamore/tcell/v3"
	"strings"
	"testing"
)

type clipboardScreen struct {
	tcell.Screen
	copied string
}

func (s *clipboardScreen) HasClipboard() bool       { return true }
func (s *clipboardScreen) SetClipboard(body []byte) { s.copied = string(body) }
func TestTextSelectionCopyDoesNotExit(t *testing.T) {
	for _, key := range []*tcell.EventKey{tcell.NewEventKey(tcell.KeyCtrlC, "", 0), tcell.NewEventKey(tcell.KeyRune, "c", tcell.ModMeta)} {
		a := fixture(t)
		screen := &clipboardScreen{Screen: a.screen}
		a.screen = screen
		a.output.Append("한글 response", false)
		a.render()
		// Use explicit screen cells so the test exercises event routing, not viewport placement.
		screen.Put(0, 2, "한", tcell.StyleDefault)
		screen.Put(2, 2, "글", tcell.StyleDefault)
		screen.Put(4, 2, "!", tcell.StyleDefault)
		a.mouse(tcell.NewEventMouse(0, 2, tcell.Button1, 0))
		a.mouse(tcell.NewEventMouse(4, 2, tcell.Button1, 0))
		a.mouse(tcell.NewEventMouse(4, 2, tcell.ButtonNone, 0))
		a.render()
		a.keyEvent(key)
		if a.exit || screen.copied != "한글!" {
			t.Fatalf("exit=%v copied=%q", a.exit, screen.copied)
		}
		if a.shortcuts()[0].Label != "복사" {
			t.Fatal("copy hint missing")
		}
		a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
		a.keyEvent(tcell.NewEventKey(tcell.KeyCtrlC, "", 0))
		if !a.exit {
			t.Fatal("unselected Ctrl+C no longer exits")
		}
	}
}
func TestTextSelectionCopiesInputAndFreezesStreaming(t *testing.T) {
	a := fixture(t)
	a.editor.Set("copy my input")
	a.render()
	row := a.inputRegion.y
	a.mouse(tcell.NewEventMouse(2, row, tcell.Button1, 0))
	a.mouse(tcell.NewEventMouse(14, row, tcell.Button1, 0))
	a.mouse(tcell.NewEventMouse(14, row, tcell.ButtonNone, 0))
	before := a.textSelection.Text()
	if !strings.Contains(before, "copy my input") {
		t.Fatalf("selection=%q", before)
	}
	a.stream = "A new streamed response"
	a.render()
	if a.textSelection.Text() != before {
		t.Fatal("streaming altered copied text")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyRune, "x", 0))
	if a.textSelection.Active() || a.editor.String() != "copy my inputx" {
		t.Fatal("typing did not clear selection and edit input")
	}
}

func TestTextSelectionClearsOnViewChange(t *testing.T) {
	a := fixture(t)
	a.render()
	a.mouse(tcell.NewEventMouse(0, 2, tcell.Button1, 0))
	a.mouse(tcell.NewEventMouse(5, 2, tcell.Button1, 0))
	a.mouse(tcell.NewEventMouse(5, 2, tcell.ButtonNone, 0))
	a.page = "model"
	a.render()
	if a.textSelection.Active() {
		t.Fatal("selection overlays another view")
	}
}
