// Purpose: Verify wrapped bottom hints resize the composer and collapse when enough width returns.
// Pattern: Virtual terminal layout regression tests.
// Usage: go test ./internal/tui/session -run TestFooterWrap.
// Related: internal/tui/session/render.go, internal/tui/view/footer.go, internal/tui/session/render_chat.go.
package session

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/v-code/internal/tui/viewport"
)

func TestFooterWrapAndCollapse(t *testing.T) {
	a := fixture(t)
	a.selection.Model = "chosen"
	a.editor.Set("draft")
	for _, test := range []struct{ width, height, status int }{
		{240, 24, 22},
		{60, 24, 21},
		{60, 12, 9},
		{240, 24, 22},
	} {
		a.screen.SetSize(test.width, test.height)
		a.render()
		if got := strings.TrimSpace(screenLine(a.screen, test.status, test.width)); got != "chosen" {
			t.Fatalf("model row at %dx%d: %q", test.width, test.height, got)
		}
		if got := strings.TrimSpace(screenLine(a.screen, test.status-1, test.width)); got != "> draft" {
			t.Fatalf("input overlapped footer at %dx%d: %q", test.width, test.height, got)
		}
		var keys strings.Builder
		for y := test.status + 1; y < test.height; y++ {
			keys.WriteString(screenLine(a.screen, y, test.width))
		}
		for _, key := range []string{"^C", "Enter", "Shift+Enter", "Ctrl/Cmd+↑↓", "PgUp/Dn", "!"} {
			if !strings.Contains(keys.String(), key) {
				t.Fatalf("missing hint %s: %s", key, keys.String())
			}
		}
		if strings.Contains(keys.String(), "…") {
			t.Fatalf("fitting hints collapsed to ellipsis: %s", keys.String())
		}
	}
}

func TestFooterWrapKeepsInputSelectionAnchored(t *testing.T) {
	for _, test := range []struct {
		text, selected, remaining string
		height, first, last       int
	}{
		{"draft text", "draft", " text", 24, 2, 6},
		{"A👩‍💻B", "👩‍💻", "AB", 24, 3, 4},
		{"first\nlast", "last", "first\n", 5, 0, 3},
	} {
		t.Run(test.selected, func(t *testing.T) {
			a := fixture(t)
			a.screen.SetSize(80, test.height)
			a.editor.Set(test.text)
			a.render()
			y := a.inputRegion.y + a.inputRegion.window.CursorRow
			a.mouse(tcell.NewEventMouse(test.first, y, tcell.Button1, 0))
			a.render()
			a.mouse(tcell.NewEventMouse(test.last, y, tcell.Button1, 0))
			a.render()
			if current := a.inputRegion.y + a.inputRegion.window.CursorRow; current != y {
				t.Fatal("input moved while the mouse button was held")
			}
			a.mouse(tcell.NewEventMouse(test.last, y, tcell.ButtonNone, 0))
			a.render()
			if got := a.textSelection.Text(); got != test.selected {
				t.Fatalf("selection changed after footer reflow: %q", got)
			}
			y = a.inputRegion.y + a.inputRegion.window.CursorRow
			_, style, _ := a.screen.Get(test.first, y)
			if style.GetAttributes()&tcell.AttrReverse == 0 {
				t.Fatal("selection highlight detached from the input")
			}
			a.keyEvent(tcell.NewEventKey(tcell.KeyBackspace, "", 0))
			if a.editor.String() != test.remaining {
				t.Fatalf("selection deleted wrong input: %q", a.editor.String())
			}
		})
	}
}

func TestFooterWrapIgnoresStaleScrollbarGeometry(t *testing.T) {
	a := fixture(t)
	a.screen.SetSize(100, 24)
	a.selection.Model = "chosen"
	a.output.Append(strings.Repeat("line\n", 18)+"line", false)
	a.scrollbar.bar = viewport.NewScrollbar(200, 18, 0)
	a.render()
	if a.scrollbar.bar.Visible() || strings.TrimSpace(screenLine(a.screen, 22, 100)) != "chosen" {
		t.Fatal("scrollbar geometry affected the keyboard footer height")
	}
}
