// Purpose: Verify preview keys, literal @ input and plain file/directory labels.
// Pattern: Terminal workflow tests.
// Usage: go test ./internal/tui/session -run TestFilePreview.
// Related: internal/tui/session/file_preview.go, internal/tui/session/files.go.
package session

import (
	"context"
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/meca/internal/fileview"
	"github.com/keonho-kim/meca/internal/tui/preview"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFilePreviewModesAndFolderNoOp(t *testing.T) {
	a := fixture(t)
	ctx, cancel := context.WithTimeout(a.ctx, 5*time.Second)
	a.ctx = ctx
	defer func() { cancel(); a.shutdown() }()
	root := a.opts.Session.ProjectRoot
	if err := os.Mkdir(filepath.Join(root, "sub"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("hello\nworld\n"), 0600); err != nil {
		t.Fatal(err)
	}
	a.openFiles(".")
	finishBrowserIO(t, a)
	a.status = ""
	renderPreview(t, a)
	if !strings.Contains(screenLine(a.screen, 3, 80), "[ ] sub/") || !strings.Contains(screenLine(a.screen, 4, 80), "[ ] a.txt") {
		t.Fatal("file types not visible")
	}
	for _, key := range []tcell.Key{tcell.KeyEnter, tcell.KeyTab} {
		a.keyEvent(tcell.NewEventKey(key, "", 0))
		if a.filePreview != nil || len(a.browser.Columns) != 1 || a.status != "" {
			t.Fatal("folder preview key had side effects")
		}
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyDown, "", 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyRune, " ", 0))
	finishBrowserIO(t, a)
	renderPreview(t, a)
	if !strings.Contains(screenLine(a.screen, 4, 80), "[x] a.txt") {
		t.Fatal("selection checkbox missing")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyTab, "", 0))
	for a.filePreview != nil && a.filePreview.Loading {
		select {
		case n := <-a.events:
			a.receive(n)
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
	}
	if a.filePreview == nil || a.filePreview.Full {
		t.Fatal("Tab did not open half preview")
	}
	renderPreview(t, a)
	if !strings.Contains(screenLine(a.screen, 2, 80)[40:], "1 | hello") {
		t.Fatal("preview not in right half")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	renderPreview(t, a)
	if !a.filePreview.Full || !strings.Contains(screenLine(a.screen, 2, 80), "1 | hello") {
		t.Fatal("Enter did not expand preview")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	if a.filePreview != nil || a.page != "files" || a.browser.Current().Index != 1 {
		t.Fatal("Esc did not restore browser")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	if a.filePreview == nil || !a.filePreview.Full {
		t.Fatal("Enter did not open fullscreen directly")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	a.page = "chat"
	typeInput(a, "@src/a")
	if a.editor.String() != "@src/a" || a.commandOpen {
		t.Fatal("@ still activated completion")
	}
}

func TestFilePreviewEscapeLeavesAgentRunning(t *testing.T) {
	a := fixture(t)
	a.previewRevision = 4
	a.page = "files"
	a.filePreview = &preview.State{Content: "hello", Full: true}
	a.worker = workerAgent
	cancelled := false
	a.taskCancel = func() { cancelled = true }
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	if a.filePreview != nil || cancelled || !a.worker.active() {
		t.Fatal("preview Esc interrupted agent")
	}
	a.receive(notice{PreviewRevision: a.previewRevision - 1, FilePreview: &fileview.Preview{Content: "stale"}})
	if a.filePreview != nil {
		t.Fatal("stale read reopened preview")
	}
}
