// Purpose: Verify explicit file-send confirmation and safe cancellation of selection-dependent sends.
// Pattern: Virtual terminal tests with controlled persistence.
// Usage: go test ./internal/tui/session -run 'TestFileSend|TestFileSelectionPending'.
// Related: internal/tui/session/file_submission.go, internal/tui/session/file_selection.go, internal/tui/session/paste.go.
package session

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/gdamore/tcell/v3"

	requestinput "github.com/keonho-kim/meca/internal/commands/input"
	"github.com/keonho-kim/meca/internal/tui/locale"
)

func TestFileSendRequiresExplicitConfirmation(t *testing.T) {
	for _, cancel := range []tcell.Key{tcell.KeyRune, tcell.KeyEscape} {
		a := fixture(t)
		a.language = locale.English
		a.page, a.files.browser.Input = "files", true
		a.compose.editor.Set("review the selected files")
		a.files.browser.Selected = map[string]bool{"src/main.py": true}
		a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
		if !a.files.fileSendConfirm || a.page != "files" || !strings.Contains(modelScreen(a), "Send this request? (y/N)") {
			t.Fatal("Enter did not ask for confirmation")
		}
		labels := keymapLabels(t, a)
		if labels["y"] != "Send" || labels["N"] != "Cancel" || labels["Esc"] != "Keep writing" || labels["Enter"] != "" {
			t.Fatal(labels)
		}
		a.pasteBoundary(true)
		a.keyEvent(tcell.NewEventKey(tcell.KeyRune, "y", 0))
		a.pasteBoundary(false)
		a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
		if !a.files.fileSendConfirm || a.compose.editor.String() != "review the selected files" || a.work.state.active() {
			t.Fatal("paste or Enter bypassed explicit confirmation")
		}
		a.keyEvent(tcell.NewEventKey(tcell.KeyRune, "n", 0))
		if !a.files.fileSendConfirm || a.page != "files" {
			t.Fatal("lowercase n dismissed send confirmation")
		}
		a.keyEvent(tcell.NewEventKey(cancel, "N", 0))
		if a.files.fileSendConfirm || a.page != "files" || a.compose.editor.String() != "review the selected files" || !a.files.browser.Selected["src/main.py"] {
			t.Fatal("cancellation changed the draft or selected paths")
		}
		a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
		if a.page != "chat" || a.compose.editor.String() != "review the selected files" {
			t.Fatal("Escape did not close the browser while preserving the draft")
		}
	}
}

func TestFileSelectionPendingSendPreservesDraft(t *testing.T) {
	for _, action := range []string{"cancel", "edit", "save-error"} {
		t.Run(action, func(t *testing.T) {
			a := fixture(t)
			if err := os.Mkdir(filepath.Join(a.opts.Session.ProjectRoot, "src"), 0700); err != nil {
				t.Fatal(err)
			}
			blocked := &selectionBarrier{Store: a.opts.Store, started: make(chan struct{}), release: make(chan struct{})}
			if action == "save-error" {
				blocked.saveErr = errors.New("selection save failed")
			}
			a.opts.Store = blocked
			var once sync.Once
			release := func() { once.Do(func() { close(blocked.release) }) }
			t.Cleanup(release)
			a.openFiles(".")
			finishBrowserIO(t, a)
			a.work.state = workerAgent
			a.compose.steering = make(chan requestinput.Request, 1)
			a.keyEvent(tcell.NewEventKey(tcell.KeyRune, " ", 0))
			<-blocked.started
			a.keyEvent(tcell.NewEventKey(tcell.KeyBacktab, "", 0))
			a.compose.editor.Set("review src")
			a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
			a.keyEvent(tcell.NewEventKey(tcell.KeyRune, "y", 0))
			if a.page != "chat" || a.compose.fileSubmission == nil {
				t.Fatal("send did not wait for persistence in chat")
			}
			if action == "cancel" {
				a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
			} else if action == "edit" {
				a.keyEvent(tcell.NewEventKey(tcell.KeyRune, " changed", 0))
			}
			release()
			finishBrowserIO(t, a)
			if a.compose.fileSubmission != nil || len(a.compose.steering) != 0 || !strings.HasPrefix(a.compose.editor.String(), "review src") || a.work.state.stopping() {
				t.Fatal("cancelled, edited or failed selection unexpectedly sent or lost the draft")
			}
			if action == "save-error" && !strings.Contains(a.status, "selection save failed") {
				t.Fatal("selection save failure was hidden")
			}
		})
	}
}

func TestFileSendWithoutModelKeepsDraft(t *testing.T) {
	a := fixture(t)
	a.page, a.files.browser.Input = "files", true
	a.compose.editor.Set("review files")
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyRune, "y", 0))
	if a.page != "chat" || a.compose.editor.String() != "review files" || a.work.state.active() || a.status == "" {
		t.Fatal("unavailable connection lost the confirmed draft")
	}
}
