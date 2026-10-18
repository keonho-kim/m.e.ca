// Purpose: Load directory pages without blocking terminal input and discard cancelled navigation results.
// Pattern: Cancellable worker with revision-bound results.
// Usage: File navigation submits requests; receiveDirectory applies them on the event loop.
// Related: internal/tui/session/files.go, internal/tui/session/events.go, internal/tui/browser/browser.go.
package session

import (
	"context"
	"fmt"
	"github.com/keonho-kim/v-code/internal/tools/local"
)

type directoryPage struct {
	revision, offset int
	mode, path       string
	entries          []local.Entry
	more             bool
	err              error
}

func (a *app) cancelDirectory() {
	if a.directoryCancel != nil {
		a.directoryCancel()
		a.directoryCancel = nil
	}
	a.directoryRevision++
	a.directoryLoading = false
}

func (a *app) requestDirectory(mode, path string, offset int) {
	a.cancelDirectory()
	ctx, cancel := context.WithCancel(a.ctx)
	a.directoryCancel = cancel
	a.directoryLoading = true
	page := directoryPage{revision: a.directoryRevision, mode: mode, path: path, offset: offset}
	executor, settings := a.opts.Tools, a.settings
	a.workers.Add(1)
	go func() {
		defer a.workers.Done()
		defer cancel()
		if mode == "open" {
			ref, err := executor.Reference(settings, path)
			page.err = err
			if err == nil && !ref.Directory {
				page.err = fmt.Errorf("not a directory: %s", path)
			}
			page.path = ref.Path
		}
		if page.err == nil {
			page.entries, page.more, page.err = executor.Directory(ctx, settings, page.path, offset, 1000)
		}
		if ctx.Err() == nil {
			a.post(ctx, notice{Directory: &page})
		}
	}()
}

func (a *app) receiveDirectory(page *directoryPage) {
	if page.revision != a.directoryRevision || a.page != "files" {
		return
	}
	a.directoryLoading = false
	a.directoryCancel = nil
	if page.err != nil {
		a.fail(page.err)
		return
	}
	a.clearSelection()
	switch page.mode {
	case "open":
		a.browser.Open(page.path, page.entries, page.more)
	case "push":
		a.browser.Push(page.path, page.entries, page.more)
	case "page":
		if column := a.browser.Current(); column != nil && column.Path == page.path {
			column.Entries, column.More, column.Offset, column.Index = page.entries, page.more, page.offset, 0
		}
	}
}
