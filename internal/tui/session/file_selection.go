// Purpose: Persist browser reference selections without blocking keyboard handling.
// Pattern: Serial cancellable storage worker.
// Usage: Space queues one selection; receive applies successful persisted state.
// Related: internal/tui/session/files.go, internal/tui/session/events.go, internal/tui/session/storage.go.
package session

import (
	"maps"
)

type fileSelectionResult struct {
	path     string
	selected bool
	err      error
}

func (a *app) selectFile(path string) {
	if !a.files.beginSelection(path) {
		return
	}
	selections := maps.Clone(a.files.browser.Selected)
	executor, settings, store, id, ctx := a.opts.Files, a.settings, a.opts.Store, a.opts.Session.ID, a.ctx
	a.workers.Add(1)
	go func() {
		defer a.workers.Done()
		ref, err := executor.Reference(ctx, settings, path)
		selected := !selections[ref.Path]
		if err == nil {
			err = store.SelectFile(ctx, id, ref.Path, selected)
		}
		a.post(ctx, notice{FileSelection: &fileSelectionResult{path: ref.Path, selected: selected, err: err}})
	}()
}

func (a *app) receiveFileSelection(result *fileSelectionResult) {
	next, queued := a.files.completeSelection(result)
	if result.err != nil {
		a.compose.fileSubmission = nil
		a.fail(result.err)
	}
	if queued {
		a.selectFile(next)
		return
	}
	a.finishFileSubmission()
}
