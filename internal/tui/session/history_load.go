// Purpose: Load older transcript pages outside the input loop and commit their ordered projections.
// Pattern: Single owned history worker.
// Usage: Scrolling at the oldest loaded record starts a read; receive applies the result.
// Related: internal/tui/session/history.go, internal/tui/session/events.go, internal/tui/viewport/viewport.go.
package session

import (
	"github.com/keonho-kim/v-code/internal/tui/transcript"
	"github.com/keonho-kim/v-code/internal/tui/viewport"
)

type historyPage struct {
	revision int
	blocks   []viewport.Block
	cursor   int64
	err      error
}

func (a *app) loadOlderHistory() {
	if a.historyLoading || a.historyEnd {
		return
	}
	a.historyLoading = true
	ctx, store, id, cursor, redact, language := a.ctx, a.opts.Store, a.opts.Session.ID, a.historyCursor, a.opts.Vault.Redact, a.language
	revision := a.historyRevision
	a.workers.Add(1)
	go func() {
		defer a.workers.Done()
		records, next, err := store.Timeline(ctx, id, cursor, 100)
		page := historyPage{revision: revision, cursor: next, err: err}
		formatter := transcript.Formatter{Language: language, Redact: redact}
		page.blocks = formatter.History(records)
		a.post(ctx, notice{History: &page})
	}()
}

func (a *app) receiveHistory(page *historyPage) {
	if page.revision != a.historyRevision {
		return
	}
	a.historyLoading = false
	if page.err != nil {
		a.fail(page.err)
		return
	}
	if page.cursor == a.historyCursor || page.cursor == 0 {
		a.historyEnd = true
	}
	a.historyCursor = page.cursor
	a.output.Prepend(page.blocks)
}
