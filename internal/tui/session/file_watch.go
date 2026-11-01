// Purpose: Refresh the visible directory and preview from one event-loop-owned filesystem watcher.
// Pattern: Active-view subscription with a coalescing one-shot timer.
// Usage: The terminal loop reconciles the active path and consumes watcher channels directly.
// Related: internal/tui/session/event_loop.go, internal/tui/session/directory.go, internal/tui/session/file_preview.go.
package session

import (
	"context"
	"errors"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

type watchRequest struct {
	path     string
	revision int
}

type watchResult struct {
	watchRequest
	watcher *fsnotify.Watcher
	err     error
}

type fileWatch struct {
	watcher             *fsnotify.Watcher
	path                string
	events              <-chan fsnotify.Event
	errors              <-chan error
	timer               *time.Timer
	ticks               <-chan time.Time
	dirty, previewDirty bool
}

func (a *app) watchPath(path string) string {
	if !filepath.IsAbs(path) {
		path = filepath.Join(a.opts.Session.ProjectRoot, path)
	}
	return filepath.Clean(path)
}

func (a *app) syncFileWatch() {
	column := a.files.browser.Current()
	if a.page != "files" || a.exit || a.ctx.Err() != nil || column == nil {
		if a.files.fileWatch.path != "" {
			a.stopFileWatch()
		}
		return
	}
	path := a.watchPath(column.Path)
	if path != a.files.fileWatch.path {
		a.stopFileWatch()
		a.files.fileWatch.path = path
		request := watchRequest{path: path, revision: a.files.watchRevision}
		if a.files.watchReads.Submit(request) {
			a.startFileWatch(request)
		}
	}
	if a.files.fileWatch.dirty && a.files.fileWatch.ticks == nil && !a.files.directoryReads.Running() {
		a.scheduleFileRefresh()
	}
}

func (a *app) stopFileWatch() {
	a.files.watchReads.Stop()
	a.files.watchRevision++
	if a.files.fileWatch.timer != nil {
		a.files.fileWatch.timer.Stop()
	}
	if a.files.fileWatch.watcher != nil {
		if err := a.files.fileWatch.watcher.Close(); err != nil {
			a.fail(err)
		}
	}
	a.files.fileWatch = fileWatch{}
}

// Registration can inspect directory entries on some platforms. Keep it off
// the terminal event loop, with one active registration and one latest pending path.
func (a *app) startFileWatch(request watchRequest) {
	ctx, cancel := context.WithCancel(a.ctx)
	a.files.watchReads.Begin(request.revision, cancel)
	a.workers.Go(func() {
		defer cancel()
		result := watchResult{watchRequest: request}
		result.watcher, result.err = fsnotify.NewWatcher()
		if result.err == nil && ctx.Err() == nil {
			result.err = result.watcher.Add(request.path)
		}
		if ctx.Err() != nil {
			result.err = ctx.Err()
		}
		if result.err != nil && result.watcher != nil {
			_ = result.watcher.Close()
			result.watcher = nil
		}
		if err := a.post(a.ctx, notice{FileWatch: &result}); err != nil && result.watcher != nil {
			_ = result.watcher.Close()
		}
	})
}

func (a *app) receiveFileWatch(result *watchResult) {
	if next := a.files.watchReads.Complete(result.revision); next != nil && !a.exit && a.ctx.Err() == nil {
		a.startFileWatch(*next)
	}
	if result.revision != a.files.watchRevision || a.page != "files" || a.exit || a.ctx.Err() != nil {
		if result.watcher != nil {
			_ = result.watcher.Close()
		}
		return
	}
	if result.err != nil {
		a.status = a.language.Format("Live file updates unavailable: %s", a.opts.Vault.Redact(result.err.Error()))
		return
	}
	a.files.fileWatch.watcher, a.files.fileWatch.events, a.files.fileWatch.errors = result.watcher, result.watcher.Events, result.watcher.Errors
	// Catch changes made between the directory read and registration completion.
	a.files.fileWatch.dirty, a.files.fileWatch.previewDirty = true, a.files.filePreview != nil
}

func (a *app) fileWatchFailed(err error) {
	if errors.Is(err, fsnotify.ErrEventOverflow) {
		a.files.fileWatch.dirty, a.files.fileWatch.previewDirty = true, a.files.filePreview != nil
		a.scheduleFileRefresh()
		return
	}
	path := a.files.fileWatch.path
	a.stopFileWatch()
	a.files.fileWatch.path = path // Retry when navigation opens a new subscription.
	if err != nil {
		a.status = a.language.Format("Live file updates unavailable: %s", a.opts.Vault.Redact(err.Error()))
	}
}

func (a *app) observeFileChange(event fsnotify.Event) {
	if event.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Remove|fsnotify.Rename) == 0 {
		return
	}
	a.files.fileWatch.dirty = true
	if a.files.filePreviewPath != "" && a.watchPath(event.Name) == a.watchPath(a.files.filePreviewPath) {
		a.files.fileWatch.previewDirty = true
	}
	a.scheduleFileRefresh()
}

func (a *app) scheduleFileRefresh() {
	const delay = 150 * time.Millisecond
	if a.files.fileWatch.timer == nil {
		a.files.fileWatch.timer = time.NewTimer(delay)
	} else {
		a.files.fileWatch.timer.Reset(delay)
	}
	a.files.fileWatch.ticks = a.files.fileWatch.timer.C
}

func (a *app) refreshWatchedFiles() {
	a.files.fileWatch.ticks = nil
	if a.page != "files" || a.exit {
		a.stopFileWatch()
		return
	}
	if a.files.directoryReads.Running() || !a.files.fileWatch.dirty {
		return
	}
	column := a.files.browser.Current()
	if column == nil || a.watchPath(column.Path) != a.files.fileWatch.path {
		return
	}
	a.files.fileWatch.dirty = false
	a.requestDirectory("refresh", column.Path, column.Offset)
	if a.files.fileWatch.previewDirty {
		a.files.fileWatch.previewDirty = false
		a.refreshFilePreview()
	}
}
