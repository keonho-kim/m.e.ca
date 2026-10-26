// Purpose: Load the active directory page and retain only the latest pending navigation.
// Pattern: Single-reader bulkhead with revision-bound completion.
// Usage: File navigation submits requests; receiveDirectory applies them on the event loop.
// Related: internal/tui/session/files.go, internal/tui/session/events.go, internal/tui/browser/browser.go.
package session

import (
	"context"
	"fmt"
	"github.com/keonho-kim/meca/internal/tools/local"
)

type directoryPage struct {
	revision, offset int
	mode, path       string
	entries          []local.Entry
	more             bool
	err              error
}

func (a *app) cancelDirectory() {
	a.directoryReads.Stop()
	a.directoryRevision++
	a.directoryLoading = false
}

func (a *app) requestDirectory(mode, path string, offset int) {
	a.cancelDirectory()
	a.directoryLoading = mode != "refresh"
	page := directoryPage{revision: a.directoryRevision, mode: mode, path: path, offset: offset}
	if a.directoryReads.Submit(page) {
		a.startDirectoryRead(page)
	}
}

func (a *app) startDirectoryRead(page directoryPage) {
	ctx, cancel := context.WithCancel(a.ctx)
	a.directoryReads.Begin(page.revision, cancel)
	path, mode := page.path, page.mode
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
		if page.err == nil && ctx.Err() == nil {
			page.entries, page.more, page.err = executor.Directory(ctx, settings, page.path, page.offset, 1000)
		}
		if ctx.Err() != nil {
			page.entries, page.err = nil, ctx.Err()
		}
		// Cancellation does not release the slot until the actual I/O returns.
		a.post(a.ctx, notice{Directory: &page})
	}()
}

func (a *app) receiveDirectory(page *directoryPage) {
	if next := a.directoryReads.Complete(page.revision); next != nil && !a.exit && a.ctx.Err() == nil {
		a.startDirectoryRead(*next)
	}
	if page.revision != a.directoryRevision || a.page != "files" {
		return
	}
	a.directoryLoading = false
	if page.err != nil {
		a.fail(page.err)
		return
	}
	if page.mode != "refresh" {
		a.clearSelection()
	}
	switch page.mode {
	case "open":
		a.browser.Open(page.path, page.entries, page.more)
	case "push":
		a.browser.Push(page.path, page.entries, page.more)
	case "page":
		if column := a.browser.Current(); column != nil && column.Path == page.path {
			column.Entries, column.More, column.Offset, column.Index = page.entries, page.more, page.offset, 0
		}
	case "refresh":
		if column := a.browser.Current(); column != nil && column.Path == page.path {
			selected := ""
			if column.Index >= 0 && column.Index < len(column.Entries) {
				selected = column.Entries[column.Index].Name
			}
			column.Entries, column.More = page.entries, page.more
			column.Index = max(0, min(column.Index, len(page.entries)-1))
			for i, entry := range page.entries {
				if entry.Name == selected {
					column.Index = i
					break
				}
			}
		}
	}
}
