// Purpose: Connect active browser navigation and refresh results to its owned file controller.
// Pattern: Event-loop adapter.
// Usage: The terminal loop selects watcher channels and applies directory/preview refresh requests.
// Related: internal/tui/browser/controller/watch.go, internal/tui/session/event_loop.go, internal/tui/session/directory.go.
package session

import (
	"github.com/fsnotify/fsnotify"
	"github.com/keonho-kim/meca/internal/tui/browser/controller"
	"path/filepath"
)

func (a *app) watchPath(path string) string {
	if !filepath.IsAbs(path) {
		path = filepath.Join(a.opts.Session.ProjectRoot, path)
	}
	return filepath.Clean(path)
}
func (a *app) syncFileWatch() {
	column := a.files.browser.Current()
	if a.page != "files" || a.exit || a.ctx.Err() != nil || column == nil {
		if a.files.reads.Watch.Path != "" {
			a.stopFileWatch()
		}
		return
	}
	a.fail(a.files.reads.SyncWatch(a.ctx, a, a.watchPath(column.Path)))
}
func (a *app) stopFileWatch() { a.fail(a.files.reads.StopWatch()) }
func (a *app) receiveFileWatch(result *controller.WatchResult) {
	if err := a.files.reads.AcceptWatch(a.ctx, a, result, a.page == "files" && !a.exit, a.files.filePreview != nil); err != nil {
		a.status = a.language.Format("Live file updates unavailable: %s", a.opts.Vault.Redact(err.Error()))
	}
}
func (a *app) fileWatchFailed(err error) {
	if err = a.files.reads.WatchFailed(err, a.files.filePreview != nil); err != nil {
		a.status = a.language.Format("Live file updates unavailable: %s", a.opts.Vault.Redact(err.Error()))
	}
}
func (a *app) observeFileChange(event fsnotify.Event) {
	path := a.files.filePreviewPath
	if path != "" {
		path = a.watchPath(path)
	}
	a.files.reads.Observe(event, path)
}
func (a *app) refreshWatchedFiles() {
	if a.page != "files" || a.exit {
		a.stopFileWatch()
		return
	}
	column := a.files.browser.Current()
	path := ""
	if column != nil {
		path = a.watchPath(column.Path)
	}
	refresh, preview := a.files.reads.Refresh(path)
	if !refresh {
		return
	}
	a.requestDirectory("refresh", column.Path, column.Offset)
	if preview {
		a.refreshFilePreview()
	}
}
