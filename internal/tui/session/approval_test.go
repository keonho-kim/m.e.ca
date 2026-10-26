// Purpose: Verify unified approval previews remain scrollable and human rejection controls execution.
// Pattern: Terminal behavior tests.
// Usage: go test ./internal/tui/session -run TestApprovalDiffNavigation.
// Related: internal/tui/session/render.go, internal/tui/session/keys.go, internal/tools/local/diff/diff.go.
package session

import (
	"github.com/gdamore/tcell/v3"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/tools/local/diff"
	"strings"
	"testing"
)

func TestApprovalDiffNavigation(t *testing.T) {
	a := fixture(t)
	a.page = "approval"
	preview, _, _, err := diff.Analyze("file", []byte(strings.Repeat("old\n", 30)), []byte(strings.Repeat("new\n", 30)), true, true)
	if err != nil {
		t.Fatal(err)
	}
	a.proposal = &toolapi.Approval{Name: "Editing", Target: "file", Diff: true, Preview: preview}
	a.choices = []string{"Approve", "Reject"}
	a.decision = make(chan bool, 1)
	a.render()
	first := screenLine(a.screen, 2, 80)
	if !strings.Contains(first, "--- file") || !strings.Contains(screenLine(a.screen, 4, 80), "@@ -") {
		t.Fatal("approval did not render a unified diff")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyPgDn, "", 0))
	a.render()
	if a.previewOffset == 0 || screenLine(a.screen, 2, 80) == first {
		t.Fatal("PageDown did not scroll the diff")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyPgUp, "", 0))
	a.render()
	if a.previewOffset != 0 || screenLine(a.screen, 2, 80) != first {
		t.Fatal("PageUp did not restore the diff start")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	if <-a.decision || a.page != "chat" {
		t.Fatal("dismissed diff became approval")
	}
}
func TestApprovalChangedRowsHaveContrastingBackgrounds(t *testing.T) {
	a := fixture(t)
	a.page = "approval"
	preview, _, _, err := diff.Analyze("file", []byte("context\n"+strings.Repeat("old", 40)+"\n"), []byte("context\n"+strings.Repeat("new", 40)+"\n"), true, true)
	if err != nil {
		t.Fatal(err)
	}
	a.proposal = &toolapi.Approval{Name: "Editing", Target: "file", Diff: true, Preview: preview}
	a.choices = []string{"Approve", "Reject"}
	a.render()
	for _, y := range []int{6, 7, 8, 9} {
		foreground, background := tcell.NewHexColor(0xe6edf3), tcell.NewHexColor(0x14532d)
		if y <= 7 {
			background = tcell.NewHexColor(0x6b2230)
		}
		for _, x := range []int{0, 79} {
			_, style, _ := a.screen.Get(x, y)
			fg, bg := style.GetForeground(), style.GetBackground()
			if bg != background || fg != foreground {
				t.Fatalf("changed cell %d,%d lacks the expected bright text and dark green/red background", x, y)
			}
		}
	}
	for _, y := range []int{2, 3, 4, 5, 11, 12} {
		_, style, _ := a.screen.Get(0, y)
		bg := style.GetBackground()
		if bg != tcell.ColorDefault {
			t.Fatalf("non-change row %d has a background", y)
		}
	}
}
