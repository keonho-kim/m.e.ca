// Purpose: Persist browser reference selections without blocking keyboard handling.
// Pattern: Serial cancellable storage worker.
// Usage: Space queues one selection; receive applies successful persisted state.
// Related: internal/tui/session/files.go, internal/tui/session/events.go, internal/tui/session/storage.go.
package session

import "maps"

type fileSelectionResult struct {
	path     string
	selected bool
	err      error
}

func (a *app) selectFile(path string) {
	if a.fileSelectionBusy {
		a.fileSelectionQueue = append(a.fileSelectionQueue, path)
		return
	}
	a.fileSelectionBusy = true
	selections := maps.Clone(a.browser.Selected)
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
	a.fileSelectionBusy = false
	if result.err != nil {
		a.fileSubmission = nil
		a.fail(result.err)
	} else {
		a.referenceRevision++
		if a.browser.Selected == nil {
			a.browser.Selected = map[string]bool{}
		}
		if result.selected {
			a.browser.Selected[result.path] = true
		} else {
			delete(a.browser.Selected, result.path)
		}
	}
	if len(a.fileSelectionQueue) > 0 {
		path := a.fileSelectionQueue[0]
		a.fileSelectionQueue[0] = ""
		a.fileSelectionQueue = a.fileSelectionQueue[1:]
		a.selectFile(path)
		return
	}
	a.fileSelectionQueue = nil
	a.finishFileSubmission()
}
