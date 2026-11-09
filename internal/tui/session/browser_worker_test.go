// Purpose: Verify directory navigation and selection commits remain asynchronous and cancellable.
// Pattern: Virtual terminal worker integration tests.
// Usage: go test ./internal/tui/session -run 'TestDirectory|TestFile'.
// Related: internal/tui/session/directory.go, internal/tui/session/file_selection.go.
package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/keonho-kim/meca/internal/tui/browser/controller"

	"github.com/gdamore/tcell/v3"
)

func finishBrowserIO(t *testing.T, a *app) {
	t.Helper()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	for a.files.directoryLoading || a.files.fileSelectionBusy {
		select {
		case notice := <-a.events:
			a.receive(notice)
		case <-timer.C:
			t.Fatal("browser worker did not finish")
		}
	}
}

func TestDirectoryCancellationDiscardsLateResult(t *testing.T) {
	a := fixture(t)
	a.openFiles(".")
	revision := a.files.directoryRevision
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	if a.files.directoryLoading {
		t.Fatal("Escape did not cancel directory request")
	}
	a.receive(notice{Directory: &controller.DirectoryPage{Revision: revision, Mode: "open", Path: "stale"}})
	if len(a.files.browser.Columns) != 0 {
		t.Fatal("cancelled request changed browser")
	}
}

func TestDirectorySelectionRoundTrip(t *testing.T) {
	a := fixture(t)
	if err := os.Mkdir(filepath.Join(a.opts.Session.ProjectRoot, "folder"), 0700); err != nil {
		t.Fatal(err)
	}
	a.openFiles(".")
	finishBrowserIO(t, a)
	for _, want := range []bool{true, false} {
		a.keyEvent(tcell.NewEventKey(tcell.KeyRune, " ", 0))
		finishBrowserIO(t, a)
		if a.files.browser.Selected["folder/"] != want {
			t.Fatal("folder selection did not toggle")
		}
	}
}
