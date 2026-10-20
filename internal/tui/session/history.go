// Purpose: Reload localized terminal history and route scrolling to the older-history worker.
// Pattern: Presentation projection over paged repository data.
// Usage: Restoration and language changes reload history; scrolling requests older pages separately.
// Related: internal/storage/sqlite/timeline.go, internal/tui/viewport/viewport.go, internal/tui/view/chat.go, internal/tui/session/keys.go, internal/tui/session/events.go.
package session

import (
	"github.com/keonho-kim/v-code/internal/tui/transcript"
	"github.com/keonho-kim/v-code/internal/tui/view"
	"github.com/keonho-kim/v-code/internal/tui/viewport"
)

func (a *app) loadHistory() error {
	a.historyRevision++
	a.historyLoading = false
	records, cursor, err := a.opts.Store.Timeline(a.ctx, a.opts.Session.ID, 0, 100)
	if err != nil {
		return err
	}
	formatter := transcript.Formatter{Language: a.language, Redact: a.opts.Vault.Redact}
	output := viewport.View{Follow: true, Collapsed: a.output.Collapsed}
	for _, block := range formatter.History(records) {
		output.AppendBlock(block)
	}
	a.output = output
	a.historyCursor = cursor
	a.historyEnd = len(records) == 0 || cursor == 0
	return nil
}
func (a *app) scroll(n, w, h int) {
	lines := a.chatLines(view.TranscriptWidth(w))
	a.output.Scroll(lines, n, h)
	if n < 0 && !a.output.Follow && a.output.Start(lines, h) == 0 {
		a.loadOlderHistory()
	}
}
