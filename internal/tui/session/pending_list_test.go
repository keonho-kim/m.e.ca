// Purpose: Verify queue rows, selection color, keyboard deletion and off-screen navigation.
// Pattern: Virtual terminal behavior tests.
// Usage: go test ./internal/tui/session -run 'TestPendingList|TestPendingEditVertical|TestPendingEditDelete'.
// Related: internal/tui/session/pending_projection.go, internal/tui/session/pending_edit_keys.go, internal/tui/view/chat.go.
package session

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/meca/internal/tui/locale"
)

func TestPendingListSelectionAndSavedOrder(t *testing.T) {
	a := fixture(t)
	a.language, a.worker = locale.English, workerAgent
	a.queued.Messages = queuedRequests("first", "second\ncontinued", "third")
	a.editor.Set("draft")
	a.render()
	if got := screenLine(a.screen, 1, 80); !strings.Contains(got, "Queued (3)") {
		t.Fatal(got)
	}
	for y, want := range map[int]string{2: "1. first", 3: "2. second", 4: "continued", 5: "3. third"} {
		if !strings.Contains(screenLine(a.screen, y, 80), want) {
			t.Fatalf("missing queue row %d: %q", y, screenLine(a.screen, y, 80))
		}
		_, style, _ := a.screen.Get(5, y)
		if !style.HasDim() {
			t.Fatal("unselected queued request was not dim")
		}
	}
	if !strings.HasPrefix(screenLine(a.screen, 4, 80), "     continued") {
		t.Fatal("multiline queue text did not align under its item")
	}
	pendingArrow(a, tcell.KeyLeft)
	a.render()
	_, selected, _ := a.screen.Get(5, 5)
	if selected.GetForeground() == tcell.ColorDefault || selected.HasDim() || !selected.HasBold() {
		t.Fatal("selected queued request did not receive a clear foreground highlight")
	}
	pendingArrow(a, tcell.KeyLeft)
	a.render()
	for _, y := range []int{3, 4} {
		_, style, _ := a.screen.Get(5, y)
		if style.GetForeground() != selected.GetForeground() || style.HasDim() {
			t.Fatal("selection did not cover the multiline queued request")
		}
	}
	_, previous, _ := a.screen.Get(5, 5)
	if !previous.HasDim() || previous.GetForeground() != tcell.ColorDefault {
		t.Fatal("previous queue selection remained highlighted")
	}
	a.editor.Set("revised second")
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	a.render()
	if !slices.Equal(queuedTexts(a.queued.Messages), []string{"first", "revised second", "third"}) || a.editor.String() != "draft" ||
		!strings.Contains(screenLine(a.screen, 3, 80), "2. revised second") {
		t.Fatal("saving changed queue order, lost the draft or left a stale list")
	}
}

func TestPendingListRedactionAndRelease(t *testing.T) {
	a := fixture(t)
	a.queued.Messages = queuedRequests("first", "new-secret-value", "third")
	a.chatLayout(80)
	a.opts.Vault.Remember("new-secret-value")
	layout := a.chatLayout(80)
	var text strings.Builder
	for _, line := range layout.Window(0, layout.Count()) {
		text.WriteString(line.Text)
	}
	if strings.Contains(text.String(), "new-secret-value") || !strings.Contains(text.String(), "[redacted]") {
		t.Fatal("queue projection did not refresh after the secret list changed")
	}
	retained := a.chatProjection.queue.Entries
	a.queued.Messages = a.queued.Messages[:1]
	a.chatLayout(80)
	for _, entry := range retained[1:] {
		if entry.Block.Text != "" {
			t.Fatal("removed queue rows retained their source or display text")
		}
	}
	a.restorePending()
	a.chatLayout(80)
	if a.chatProjection.queue.Entries != nil || retained[0].Block.Text != "" {
		t.Fatal("empty queue retained projected rows")
	}
}

func TestPendingEditVerticalNavigation(t *testing.T) {
	a := fixture(t)
	a.worker = workerAgent
	a.queued.Messages = queuedRequests("first", "second", "third")
	a.editor.Set("draft")
	for _, want := range []string{"third", "second", "first", "first"} {
		pendingArrow(a, tcell.KeyUp)
		if a.editor.String() != want {
			t.Fatalf("selected=%q; want %q", a.editor.String(), want)
		}
	}
	for _, want := range []string{"second", "third", "draft"} {
		pendingArrow(a, tcell.KeyDown)
		if a.editor.String() != want {
			t.Fatalf("selected=%q; want %q", a.editor.String(), want)
		}
	}
}

func TestPendingListNavigationRevealsSelectedRequest(t *testing.T) {
	a := fixture(t)
	a.language, a.worker = locale.English, workerAgent
	a.screen.SetSize(60, 16)
	for i := range 40 {
		a.queued.Messages = append(a.queued.Messages, queuedRequests(fmt.Sprintf("request %02d", i+1))...)
	}
	a.render()
	for i := 39; i >= 0; i-- {
		pendingArrow(a, tcell.KeyLeft)
		a.render()
		found := false
		for y := 1; y <= a.scrollbar.bar.Height; y++ {
			if strings.Contains(screenLine(a.screen, y, 60), fmt.Sprintf("%d. request %02d", i+1, i+1)) {
				_, style, _ := a.screen.Get(5, y)
				found = !style.HasDim() && style.GetForeground() != tcell.ColorDefault
			}
		}
		if !found {
			t.Fatalf("queue selection %d was hidden outside the viewport", i+1)
		}
	}
}

func TestPendingListKeepsItsScrollAnchorDuringOutput(t *testing.T) {
	a := fixture(t)
	a.worker = workerAgent
	a.screen.SetSize(60, 16)
	for i := range 40 {
		a.queued.Messages = append(a.queued.Messages, queuedRequests(fmt.Sprintf("request %02d", i+1))...)
	}
	a.render()
	for range 20 {
		pendingArrow(a, tcell.KeyLeft)
	}
	a.render()
	before := screenLine(a.screen, 1, 60)
	for range 20 {
		a.output.Append("new tool output", false)
	}
	a.thinking = "streaming thought"
	a.render()
	if after := screenLine(a.screen, 1, 60); after != before {
		t.Fatalf("new output moved the queue reading anchor: before=%q after=%q", before, after)
	}
	// A deliberate scroll back into history must remain there as output arrives.
	a.scroll(-100, 60, a.scrollbar.bar.Height)
	a.render()
	before = screenLine(a.screen, 1, 60)
	a.output.Append("later tool output", false)
	a.render()
	if after := screenLine(a.screen, 1, 60); after != before {
		t.Fatalf("queue editing overrode manual history scrolling: before=%q after=%q", before, after)
	}
}

func TestPendingListResizeKeepsSelectedRequestVisible(t *testing.T) {
	a := fixture(t)
	a.worker = workerAgent
	a.screen.SetSize(80, 24)
	for i := range 20 {
		a.queued.Messages = append(a.queued.Messages, queuedRequests(fmt.Sprintf("request %02d", i+1))...)
	}
	for range 10 {
		pendingArrow(a, tcell.KeyLeft)
	}
	a.render()
	for _, size := range [][2]int{{40, 10}, {24, 8}, {80, 24}} {
		a.screen.SetSize(size[0], size[1])
		if err := a.terminalEvent(tcell.NewEventResize(size[0], size[1])); err != nil {
			t.Fatal(err)
		}
		a.render()
		visible := false
		for _, line := range a.chatVisible {
			visible = visible || line.ID == a.chatProjection.focusedID
		}
		if !visible {
			t.Fatalf("selected request hidden after resize to %dx%d", size[0], size[1])
		}
	}
}

func TestPendingEditDeleteShortcutWithTextSelection(t *testing.T) {
	for _, modifier := range []tcell.ModMask{tcell.ModCtrl, tcell.ModMeta} {
		a := fixture(t)
		a.worker = workerAgent
		a.queued.Messages = queuedRequests("first", "second", "third")
		a.editor.Set("draft")
		pendingArrow(a, tcell.KeyLeft)
		pendingArrow(a, tcell.KeyLeft)
		a.render()
		drag(a, 2, a.inputRegion.y, 4, a.inputRegion.y)
		a.keyEvent(tcell.NewEventKey(tcell.KeyBackspace, "", modifier))
		a.render()
		if !slices.Equal(queuedTexts(a.queued.Messages), []string{"first", "third"}) || a.queued.Edit != nil || a.editor.String() != "draft" || a.textSelection.Active() {
			t.Fatal("delete shortcut only edited selected text or reordered the surviving queue")
		}
		if labels := keymapLabels(t, a); labels["Enter"] != a.language.Text("Steer") || labels["Ctrl/Cmd+Backspace"] != "" {
			t.Fatal("queue editing keymap remained after deletion", labels)
		}
	}
}
