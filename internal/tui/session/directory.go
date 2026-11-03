// Purpose: Load the active directory page and retain only the latest pending navigation.
// Pattern: Single-reader bulkhead with revision-bound completion.
// Usage: File navigation submits requests; receiveDirectory applies them on the event loop.
// Related: internal/tui/session/files.go, internal/tui/session/events.go, internal/tui/browser/browser.go.
package session

import (
	"github.com/keonho-kim/meca/internal/tui/browser/controller"
)

func (a *app) requestDirectory(mode, path string, offset int) {
	page, start := a.files.planDirectory(mode, path, offset)
	if start {
		a.startDirectoryRead(page)
	}
}

func (a *app) startDirectoryRead(page controller.DirectoryPage) {
	a.files.reads.ReadDirectory(a.ctx, a, a.opts.Files, a.settings, page)
}

func (a *app) receiveDirectory(page *controller.DirectoryPage) {
	if next := a.files.reads.Directory.Complete(page.Revision); next != nil && !a.exit && a.ctx.Err() == nil {
		a.startDirectoryRead(*next)
	}
	if page.Revision != a.files.directoryRevision || a.page != "files" {
		return
	}
	a.files.directoryLoading = false
	if page.Err != nil {
		a.fail(page.Err)
		return
	}
	if page.Mode != "refresh" {
		a.clearSelection()
	}
	a.files.applyDirectory(page)
}
