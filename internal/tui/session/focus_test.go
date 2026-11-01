// Purpose: Keep displayed shortcuts and actual keyboard ownership aligned across modal overlays.
// Pattern: Focus-priority behavior regression tests.
// Usage: Run TestFocusPreview and TestFocusApproval with the virtual terminal fixture.
// Related: internal/tui/session/focus.go, internal/tui/session/keys.go, internal/tui/session/keymap.go.
package session

import (
	"testing"

	"github.com/gdamore/tcell/v3"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/tui/preview"
)

func TestFocusPreviewOwnsKeysWhileComposerRemainsOpen(t *testing.T) {
	a := screenFixture(t)
	a.page, a.files.browser.Input = "files", true
	a.compose.editor.Set("preserved request")
	a.files.filePreview = &preview.State{Path: "README.md", Content: "**body**"}
	if labels := keymapLabels(t, a); labels["Shift+R"] == "" || labels["Esc"] != "닫기" {
		t.Fatal("preview shortcuts lost their focus", labels)
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyRune, "R", tcell.ModShift))
	if !a.files.filePreview.Raw || a.compose.editor.String() != "preserved request" {
		t.Fatal("preview shortcut changed the composer")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	if a.files.filePreview != nil || a.page != "files" || !a.files.browser.Input {
		t.Fatal("closing preview discarded the underlying composer")
	}
	if labels := keymapLabels(t, a); labels["Shift+R"] != "" || labels["Shift+Tab"] != "탐색" {
		t.Fatal("composer shortcuts were not restored", labels)
	}
}

func TestFocusApprovalEscRejectsWithoutStoppingTheWorker(t *testing.T) {
	a := screenFixture(t)
	a.page, a.files.browser.Input = "files", true
	a.work.state = workerAgent
	a.decision = make(chan bool, 1)
	a.openApproval(&toolapi.Approval{Name: "Transfer"})
	if labels := keymapLabels(t, a); labels["Esc"] != "거부" {
		t.Fatal("approval shortcuts were replaced by worker controls", labels)
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	select {
	case approved := <-a.decision:
		if approved {
			t.Fatal("Esc approved the operation")
		}
	default:
		t.Fatal("Esc did not deliver the rejection")
	}
	if a.page != "files" || a.work.state != workerAgent || a.proposal != nil {
		t.Fatal("approval dismissal changed the underlying worker or browser")
	}
}
