// Purpose: Verify queued-message navigation, editing and cancellation preserve input and FIFO ownership.
// Pattern: Virtual terminal state transition tests.
// Usage: go test ./internal/tui/session -run TestPendingEdit.
// Related: internal/tui/session/pending_edit.go, internal/tui/session/pending_edit_keys.go, internal/tui/session/pending_edit_view.go.
package session

import (
	"context"
	"github.com/keonho-kim/meca/internal/tui/composer"
	"slices"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/meca/internal/tui/locale"
	"github.com/keonho-kim/meca/internal/tui/view"
)

func pendingArrow(a *app, key tcell.Key) {
	a.keyEvent(tcell.NewEventKey(key, "", tcell.ModShift))
}

func TestPendingEditNavigationPreservesDraft(t *testing.T) {
	a := screenFixture(t)
	a.worker = workerAgent
	a.queued.Messages = queuedRequests("first", "second", "third")
	a.editor.Set("unsent draft")
	a.editor.MoveTo(2)
	for _, want := range []string{"third", "second", "first", "first"} {
		pendingArrow(a, tcell.KeyLeft)
		if a.editor.String() != want {
			t.Fatalf("previous queued input=%q; want %q", a.editor.String(), want)
		}
	}
	for _, want := range []string{"second", "third", "unsent draft"} {
		pendingArrow(a, tcell.KeyRight)
		if a.editor.String() != want {
			t.Fatalf("next queued input=%q; want %q", a.editor.String(), want)
		}
	}
	if a.editor.Cursor != 2 || !slices.Equal(queuedTexts(a.queued.Messages), []string{"first", "second", "third"}) {
		t.Fatal("navigation changed the draft cursor or queue")
	}
}

func TestPendingEditSavePreservesPosition(t *testing.T) {
	for name, key := range map[string]tcell.Key{"enter": tcell.KeyEnter, "tab": tcell.KeyTab} {
		t.Run(name, func(t *testing.T) {
			a := screenFixture(t)
			a.worker = workerAgent
			a.steering = make(chan composer.Request, 1)
			a.queued.Messages = queuedRequests("first", "second", "third")
			a.editor.Set("unsent draft")
			pendingArrow(a, tcell.KeyLeft)
			pendingArrow(a, tcell.KeyLeft)
			a.keyEvent(tcell.NewEventKey(tcell.KeyCtrlU, "", 0))
			a.keyEvent(tcell.NewEventKey(tcell.KeyRune, "edited second", 0))
			a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", tcell.ModShift))
			a.keyEvent(tcell.NewEventKey(tcell.KeyRune, "line", 0))
			if a.queued.Messages[1].Text != "second" {
				t.Fatal("typing changed the queued original before saving")
			}
			a.keyEvent(tcell.NewEventKey(key, "", 0))
			if a.queued.Edit != nil || a.editor.String() != "unsent draft" ||
				!slices.Equal(queuedTexts(a.queued.Messages), []string{"first", "edited second\nline", "third"}) || len(a.steering) != 0 {
				t.Fatal("saving submitted a direction change, duplicated input or reordered the queue")
			}
		})
	}
}

func TestPendingEditCancelAndDelete(t *testing.T) {
	for _, event := range []*tcell.EventKey{
		tcell.NewEventKey(tcell.KeyEscape, "", 0),
		tcell.NewEventKey(tcell.KeyBackspace, "", tcell.ModCtrl),
		tcell.NewEventKey(tcell.KeyBackspace2, "", tcell.ModMeta),
	} {
		a := screenFixture(t)
		ctx, cancel := context.WithCancel(a.ctx)
		t.Cleanup(cancel)
		a.worker, a.taskCancel = workerAgent, cancel
		a.queued.Messages = queuedRequests("first", "second")
		a.editor.Set("unsent draft")
		pendingArrow(a, tcell.KeyLeft)
		a.keyEvent(tcell.NewEventKey(tcell.KeyRune, " changed", 0))
		a.keyEvent(event)
		want := []string{"first"}
		if event.Key() == tcell.KeyEscape {
			want = append(want, "second")
		}
		if a.queued.Edit != nil || a.editor.String() != "unsent draft" || !slices.Equal(queuedTexts(a.queued.Messages), want) || ctx.Err() != nil || a.worker.stopping() {
			t.Fatal("editing cancellation or reservation deletion stopped the running agent or lost the draft")
		}
		if event.Key() == tcell.KeyEscape {
			a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
			if ctx.Err() != context.Canceled || !a.worker.stopping() {
				t.Fatal("Esc did not stop the agent after leaving queue editing")
			}
		}
	}
}

func TestPendingEditRequiresSaveBeforeNavigation(t *testing.T) {
	a := screenFixture(t)
	a.queued.Messages = queuedRequests("first", "second")
	a.worker = workerAgent
	pendingArrow(a, tcell.KeyLeft)
	a.keyEvent(tcell.NewEventKey(tcell.KeyRune, " changed", 0))
	pendingArrow(a, tcell.KeyLeft)
	if a.queued.Edit.Index != 1 || a.editor.String() != "second changed" || a.stopNotice == "" {
		t.Fatal("navigation lost unsaved changes")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyCtrlU, "", 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	if a.queued.Edit == nil || a.queued.Messages[1].Text != "second" || a.stopNotice == "" {
		t.Fatal("empty edit deleted a reservation implicitly")
	}
}

func TestPendingEditPasteAndCommandsStayLiteral(t *testing.T) {
	a := screenFixture(t)
	a.worker = workerAgent
	a.queued.Messages = queuedRequests("queued")
	a.editor.Set("!model")
	a.refreshHints()
	pendingArrow(a, tcell.KeyLeft)
	a.keyEvent(tcell.NewEventKey(tcell.KeyCtrlU, "", 0))
	a.pasteBoundary(true)
	a.keyEvent(tcell.NewEventKey(tcell.KeyRune, "!quit", 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyRune, "body", 0))
	a.pasteBoundary(false)
	if a.commandOpen || a.exit || a.queued.Edit == nil || a.editor.String() != "!quit\nbody" {
		t.Fatal("paste invoked a command or left queued editing")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	if a.exit || a.queued.Messages[0].Text != "!quit\nbody" || a.editor.String() != "!model" || !a.commandOpen {
		t.Fatal("saving interpreted queued text as a local command or lost the normal input")
	}
	pendingArrow(a, tcell.KeyLeft)
	a.keyEvent(tcell.NewEventKey(tcell.KeyCtrlU, "", 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyRune, "!new", 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	if a.exit || a.newSession || a.queued.Messages[0].Text != "!new" {
		t.Fatal("queued editing executed a new-session command")
	}
}

func TestPendingEditKeymapAndNotice(t *testing.T) {
	for _, language := range []locale.Language{locale.Korean, locale.English} {
		a := screenFixture(t)
		a.language, a.worker = language, workerAgent
		a.queued.Messages = queuedRequests("first", "second", "third")
		if labels := keymapLabels(t, a); labels["Shift+↑/←"] != language.Text("Edit queue") || labels["Ctrl/Cmd+Backspace"] != "" {
			t.Fatal(labels)
		}
		pendingArrow(a, tcell.KeyLeft)
		labels := keymapLabels(t, a)
		for key, label := range map[string]string{"Enter": "Save edit", "Esc": "Cancel edit", "Ctrl/Cmd+Backspace": "Cancel reservation", "Shift+↑↓/←→": "Browse queue"} {
			if labels[key] != language.Text(label) {
				t.Fatalf("key %s: %v", key, labels)
			}
		}
		if labels["!"] != "" || labels["Ctrl/Cmd+↑↓"] != "" || labels["^D"] != "" {
			t.Fatal("queue editing advertised normal command or input-history actions")
		}
		for _, width := range []int{80, 160} {
			a.screen.SetSize(width, 24)
			a.render()
			if !strings.Contains(screenLine(a.screen, a.inputRegion.y-1, width), language.Format("Editing queued request %d / %d", 3, 3)) || a.footerRows-1 > view.MaxKeymapRows {
				t.Fatalf("width=%d notice=%q footer=%q rows=%d", width, screenLine(a.screen, a.inputRegion.y-1, width), footerKeyText(a), a.footerRows)
			}
			for _, key := range []string{"Esc", "Ctrl/Cmd+Backspace", "Enter", "Shift+↑↓/←→"} {
				if !strings.Contains(footerKeyText(a), key) {
					t.Fatalf("width=%d key=%s footer=%q", width, key, footerKeyText(a))
				}
			}
		}
		a.keyEvent(tcell.NewEventKey(tcell.KeyRune, " changed", 0))
		if keymapLabels(t, a)["Shift+↑↓/←→"] != "" {
			t.Fatal("dirty editing advertised navigation before save or cancellation")
		}
	}
}
