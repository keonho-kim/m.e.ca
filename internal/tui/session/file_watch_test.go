// Purpose: Verify filesystem updates preserve browser focus and release active-view subscriptions.
// Pattern: Native event integration tests with bounded waits.
// Usage: go test ./internal/tui/session -run TestFileWatch.
// Related: internal/tui/session/file_watch.go, internal/tui/session/directory.go, internal/tui/session/file_preview.go.
package session

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
)

func TestFileWatchRefreshPreservesInputSelectionAndPreview(t *testing.T) {
	a := fixture(t)
	root := a.opts.Session.ProjectRoot
	path := filepath.Join(root, "b.txt")
	if err := os.WriteFile(path, []byte("before\nsecond\n"), 0600); err != nil {
		t.Fatal(err)
	}
	a.openFiles(".")
	finishBrowserIO(t, a)
	a.openFilePreview(false)
	finishVisibleReads(t, a)
	a.files.filePreview.Offset = 1
	a.files.browser.Input = true
	a.compose.editor.Set("keep this request")
	a.syncFileWatch()
	finishVisibleReads(t, a)
	watcher := a.files.reads.Watch.Watcher
	if watcher == nil || len(watcher.WatchList()) != 1 {
		t.Fatal("visible directory was not watched")
	}
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("new"), 0600); err != nil {
		t.Fatal(err)
	}
	waitFileChange(t, a, "a.txt")
	timer := a.files.reads.Watch.Timer
	if err := os.WriteFile(path, []byte("after\nsecond\n"), 0600); err != nil {
		t.Fatal(err)
	}
	waitFileChange(t, a, "b.txt")
	if timer != a.files.reads.Watch.Timer || !a.files.reads.Watch.PreviewDirty {
		t.Fatal("updates were not coalesced")
	}
	a.refreshWatchedFiles()
	finishVisibleReads(t, a)
	if target, _ := a.files.browser.Target(); a.watchPath(target) != path {
		t.Fatalf("selection moved: %s", target)
	}
	if len(a.files.browser.Current().Entries) != 2 || a.compose.editor.String() != "keep this request" || !a.files.browser.Input {
		t.Fatal("refresh changed input or lost entries")
	}
	if a.files.filePreview.Content != "after\nsecond\n" || a.files.filePreview.Offset != 1 || a.files.filePreview.Full {
		t.Fatalf("preview state lost: %+v", a.files.filePreview)
	}
	a.syncFileWatch()
	if a.files.reads.Watch.Watcher != watcher {
		t.Fatal("refresh recreated the subscription")
	}
	a.fileWatchFailed(fsnotify.ErrEventOverflow)
	if a.files.reads.Watch.Watcher != watcher || !a.files.reads.Watch.Dirty || !a.files.reads.Watch.PreviewDirty {
		t.Fatal("overflow did not request a fresh observation")
	}
	a.closeFiles()
	if a.files.reads.Watch.Watcher != nil || a.files.reads.Watch.Ticks != nil || a.files.filePreviewPath != "" {
		t.Fatal("closed browser retained resources")
	}
	if err := watcher.Add(root); !errors.Is(err, fsnotify.ErrClosed) {
		t.Fatalf("watcher still open: %v", err)
	}
}

func TestFileWatchNavigationAndFailureRelease(t *testing.T) {
	a := fixture(t)
	root := a.opts.Session.ProjectRoot
	child := filepath.Join(root, "child")
	if err := os.Mkdir(child, 0700); err != nil {
		t.Fatal(err)
	}
	a.openFiles(".")
	finishBrowserIO(t, a)
	a.syncFileWatch()
	finishVisibleReads(t, a)
	previous := a.files.reads.Watch.Watcher
	a.requestDirectory("push", child, 0)
	finishBrowserIO(t, a)
	a.syncFileWatch()
	finishVisibleReads(t, a)
	if previous == a.files.reads.Watch.Watcher || len(a.files.reads.Watch.Watcher.WatchList()) != 1 {
		t.Fatal("navigation retained the wrong watch")
	}
	if err := previous.Add(root); !errors.Is(err, fsnotify.ErrClosed) {
		t.Fatal(err)
	}
	a.fileWatchFailed(errors.New("notification unavailable"))
	a.syncFileWatch()
	if a.files.reads.Watch.Watcher != nil || a.files.reads.Watch.Events != nil {
		t.Fatal("failed watch was retried on each frame")
	}
	a.closeFiles()
	a.openFiles(child)
	finishBrowserIO(t, a)
	a.syncFileWatch()
	finishVisibleReads(t, a)
	if a.files.reads.Watch.Watcher == nil {
		t.Fatal("reopening did not establish a new subscription")
	}
	a.requestExit()
	if a.files.reads.Watch.Watcher != nil || a.files.reads.Watch.Path != "" {
		t.Fatal("exit retained a subscription")
	}
}

func waitFileChange(t *testing.T, a *app, name string) {
	t.Helper()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	for {
		select {
		case event, ok := <-a.files.reads.Watch.Events:
			if !ok {
				t.Fatal("watch ended before the change")
			}
			a.observeFileChange(event)
			if filepath.Base(event.Name) == name && event.Op&(fsnotify.Create|fsnotify.Write) != 0 {
				return
			}
		case err := <-a.files.reads.Watch.Errors:
			t.Fatal(err)
		case <-timer.C:
			t.Fatalf("no change notification for %s", name)
		}
	}
}

func TestFileWatchCancelledRegistrationDoesNotReopen(t *testing.T) {
	a := fixture(t)
	a.openFiles(".")
	finishBrowserIO(t, a)
	a.syncFileWatch()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	select {
	case n := <-a.events:
		if n.FileWatch == nil || n.FileWatch.Watcher == nil {
			t.Fatalf("registration failed: %+v", n.FileWatch)
		}
		watcher := n.FileWatch.Watcher
		a.closeFiles()
		a.receive(n)
		if a.page != "chat" || a.files.reads.Watch.Watcher != nil || a.files.reads.WatchReads.Running() {
			t.Fatal("late registration reopened the browser")
		}
		if err := watcher.Add(a.opts.Session.ProjectRoot); !errors.Is(err, fsnotify.ErrClosed) {
			t.Fatalf("late watcher leaked: %v", err)
		}
	case <-timer.C:
		t.Fatal("registration did not finish")
	}
}

func finishVisibleReads(t *testing.T, a *app) {
	t.Helper()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	for {
		if p := a.files.filePreview; p != nil {
			width, _ := a.screen.Size()
			if !p.Full {
				width -= width/2 + 1
			}
			a.ensurePreviewLayout(width)
		}
		if !a.files.reads.Directory.Running() && !a.files.reads.Preview.Running() && !a.files.reads.WatchReads.Running() && !a.files.reads.Layout.Running() {
			return
		}
		select {
		case n := <-a.events:
			a.receive(n)
		case <-timer.C:
			t.Fatal("visible reads did not finish")
		}
	}
}

func renderPreview(t *testing.T, a *app) {
	t.Helper()
	a.render()
	finishVisibleReads(t, a)
	a.render()
}
