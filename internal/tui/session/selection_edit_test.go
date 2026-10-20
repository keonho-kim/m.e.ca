// Purpose: Verify source-backed selection copies and deletes exact input without changing transcript records.
// Pattern: Virtual terminal interaction tests.
// Usage: go test ./internal/tui/session -run 'TestInputSelection|TestTranscriptSelection|TestFieldSelection'.
// Related: internal/tui/session/text_selection.go, internal/tui/session/input_region.go.
package session

import (
	"github.com/gdamore/tcell/v3"
	"strings"
	"testing"
)

func drag(a *app, x1, y1, x2, y2 int) {
	a.mouse(tcell.NewEventMouse(x1, y1, tcell.Button1, 0))
	a.mouse(tcell.NewEventMouse(x2, y2, tcell.Button1, 0))
	a.mouse(tcell.NewEventMouse(x2, y2, tcell.ButtonNone, 0))
}

func TestInputSelectionDeletesWholeGrapheme(t *testing.T) {
	a := fixture(t)
	a.editor.Set("A👩‍💻B a\u0301Z")
	a.render()
	y := a.inputRegion.y
	drag(a, 4, y, 3, y)
	a.keyEvent(tcell.NewEventKey(tcell.KeyBackspace, "", 0))
	if a.editor.String() != "AB a\u0301Z" || a.editor.Cursor != 1 || a.textSelection.Active() {
		t.Fatalf("selection deletion: %q", a.editor.String())
	}
}

func TestInputSelectionPreservesSoftWrapsAndSpaces(t *testing.T) {
	a := fixture(t)
	a.screen.SetSize(16, 18)
	want := "abcdefghi 👩‍💻 xyz\n  tail  "
	a.editor.Set(want)
	a.render()
	r := a.inputRegion
	drag(a, 0, r.y, 15, r.y+len(r.window.Rows)-1)
	if got := a.textSelection.Text(); got != want {
		t.Fatalf("copy %q, want %q", got, want)
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyBackspace, "", 0))
	if a.editor.String() != "" {
		t.Fatal("wrapped selection was not deleted")
	}
}

func TestInputSelectionCopyThenTypingClearsHighlight(t *testing.T) {
	a := fixture(t)
	a.screen = &clipboardScreen{Screen: a.screen}
	a.editor.Set("draft text")
	a.render()
	y := a.inputRegion.y
	drag(a, 2, y, 6, y)
	a.keyEvent(tcell.NewEventKey(tcell.KeyCtrlC, "", 0))
	a.stream = "new response while selecting"
	a.render()
	if !strings.Contains(screenLine(a.screen, 1, 80), "new response") {
		t.Fatal("selection froze unrelated screen content")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyRune, "x", 0))
	a.render()
	if a.textSelection.Active() || a.editor.String() != "draft textx" {
		t.Fatal("typing retained selection")
	}
	for x := 2; x <= 6; x++ {
		_, style, _ := a.screen.Get(x, y)
		if style.GetAttributes()&tcell.AttrReverse != 0 {
			t.Fatal("highlight persisted")
		}
	}
}

func TestTranscriptSelectionCopiesSoftWrapsWithoutEditingDraft(t *testing.T) {
	a := fixture(t)
	a.screen.SetSize(16, 24)
	want := "  long line with spaces   \nnext  "
	a.output.Append(want, false)
	a.editor.Set("keep draft")
	a.render()
	lines := a.chatVisible
	drag(a, 0, 1, 15, len(lines))
	if got := a.textSelection.Text(); got != want {
		t.Fatalf("copy %q, want %q", got, want)
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyBackspace, "", 0))
	if a.editor.String() != "keep draft" || a.output.Blocks[0].Text != want {
		t.Fatal("transcript selection changed text")
	}
}

func TestFieldSelectionDeletesSelectedEndpoint(t *testing.T) {
	a := fixture(t)
	a.page = "endpoint"
	a.menuInput.Set("localhost")
	a.render()
	drag(a, 2, 2, 6, 2)
	a.keyEvent(tcell.NewEventKey(tcell.KeyBackspace, "", 0))
	if a.menuInput.String() != "host" {
		t.Fatal(a.menuInput.String())
	}
}

func TestInputSelectionRejectsUnpaintedRevision(t *testing.T) {
	a := fixture(t)
	a.editor.Set("old text")
	a.render()
	y := a.inputRegion.y
	a.editor.Set("new unsent draft")
	drag(a, 2, y, 5, y)
	a.keyEvent(tcell.NewEventKey(tcell.KeyBackspace, "", 0))
	if a.editor.String() != "new unsent draft" {
		t.Fatal("stale screen selection edited newer input")
	}
}

func TestTranscriptSelectionKeepsItsVisibleRows(t *testing.T) {
	a := fixture(t)
	a.output.Append("keep this source", false)
	a.render()
	drag(a, 0, 1, 15, 1)
	a.stream = strings.Repeat("new streamed line\n\n", 50)
	a.render()
	if !strings.Contains(screenLine(a.screen, 1, 80), "keep this source") {
		t.Fatal("stream moved selected transcript")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	a.render()
	if strings.Contains(screenLine(a.screen, 1, 80), "keep this source") {
		t.Fatal("selection dismissal did not resume projection")
	}
}
