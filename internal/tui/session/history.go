// Purpose: Reload localized terminal history and route scrolling to the older-history worker.
// Pattern: Presentation projection over paged repository data.
// Usage: Initial restoration reads directly; the settings worker prepares localized history for event-loop adoption.
// Related: internal/storage/sqlite/timeline.go, internal/tui/viewport/viewport.go, internal/tui/view/chat.go, internal/tui/session/keys.go, internal/tui/session/events.go.
package session

import (
	"context"
	"github.com/keonho-kim/meca/internal/tui/locale"
	"github.com/keonho-kim/meca/internal/tui/transcript"
	"github.com/keonho-kim/meca/internal/tui/view"
	"github.com/keonho-kim/meca/internal/tui/viewport"
)

func (a *app) loadHistory() error {
	page, err := readLocalizedHistory(a.ctx, a.opts.Store, a.opts.Session.ID, a.language, a.opts.Vault.Redact)
	if err != nil {
		return err
	}
	a.applyHistory(page)
	return nil
}

func readLocalizedHistory(ctx context.Context, store Store, id string, language locale.Language, redact func(string) string) (*historyPage, error) {
	records, cursor, err := terminalHistory(ctx, store, id, 0, 100)
	if err != nil {
		return nil, err
	}
	return &historyPage{cursor: cursor, blocks: (transcript.Formatter{Language: language, Redact: redact}).History(records)}, nil
}

func (a *app) applyHistory(page *historyPage) {
	a.cancelToolDetail()
	a.chat.historyRevision++
	a.chat.historyLoading = false
	output := viewport.View{Follow: true, Collapsed: a.chat.output.Collapsed}
	for _, block := range page.blocks {
		output.AppendBlock(block)
	}
	a.chat.output = output
	a.chat.historyCursor = page.cursor
	a.chat.historyEnd = page.cursor == 0
}
func (a *app) scroll(n, w, h int) {
	transcript := a.chatLayout(view.TranscriptWidth(w))
	transcript.Scroll(&a.chat.output, n, h)
	if n < 0 && !a.chat.output.Follow && transcript.Start(&a.chat.output, h) == 0 {
		a.loadOlderHistory()
	}
}
