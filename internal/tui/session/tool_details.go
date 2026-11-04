// Purpose: Load one visible expanded tool result at a time and release detail bodies on collapse.
// Pattern: Owned cancellable read worker with event-loop result admission.
// Usage: Chat rendering requests a missing detail; notices apply only to the same source and language revision.
// Related: internal/tui/viewport/details.go, internal/storage/sqlite/tool_detail.go.
package session

import (
	"context"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/tui/transcript"
	"github.com/keonho-kim/meca/internal/tui/viewport"
)

type detailRead struct {
	loading bool
	cancel  context.CancelFunc
}
type detailResult struct {
	id, revision int
	reference    runrecord.EventReference
	block        viewport.Block
	err          error
}

func (a *app) loadToolDetail(layout *viewport.Transcript) {
	if a.chat.output.Collapsed || a.chat.details.loading {
		return
	}
	id, ref := layout.PendingDetail()
	if ref == nil {
		return
	}
	store, ok := a.opts.Store.(interface {
		ToolEvent(context.Context, runrecord.EventReference) (runrecord.Event, error)
	})
	if !ok {
		return
	}
	ctx, cancel := context.WithCancel(a.ctx)
	a.chat.details = detailRead{loading: true, cancel: cancel}
	result := detailResult{id: id, revision: a.chat.historyRevision, reference: *ref}
	language, redact := a.language, a.opts.Vault.Redact
	a.workers.Add(1)
	go func() {
		defer a.workers.Done()
		defer cancel()
		event, err := store.ToolEvent(ctx, result.reference)
		result.err = err
		if err == nil {
			event.Reference = nil
			event.Agent = ""
			blocks := (transcript.Formatter{Language: language, Redact: redact}).Event(event)
			if len(blocks) > 0 {
				result.block = blocks[0]
			}
		}
		a.post(a.ctx, notice{Detail: &result})
	}()
}
func (a *app) receiveToolDetail(result *detailResult) {
	a.chat.details = detailRead{}
	if result.revision != a.chat.historyRevision || a.chat.output.Collapsed {
		return
	}
	if result.err != nil {
		result.block.Text = "\n  " + a.language.Text("Failed: ") + a.opts.Vault.Redact(result.err.Error()) + "\n"
	}
	a.chat.output.UpdateDetails(result.id, result.reference, result.block)
}
