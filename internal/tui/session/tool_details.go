// Purpose: Load one visible expanded tool result at a time and release detail bodies on collapse.
// Pattern: Owned cancellable read worker with event-loop result admission.
// Usage: Chat rendering requests a missing detail; notices apply only to the same source and language revision.
// Related: internal/tui/viewport/details.go, internal/storage/sqlite/tool_detail.go.
package session

import (
	"context"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/storage/history"
	"github.com/keonho-kim/meca/internal/tui/transcript"
	"github.com/keonho-kim/meca/internal/tui/viewport"
)

type detailRead struct {
	cancel    context.CancelFunc
	abandoned bool
}
type detailResult struct {
	id, revision int
	reference    runrecord.EventReference
	block        viewport.Block
	err          error
}

func (a *app) loadToolDetail(layout *viewport.Transcript) {
	if a.chat.output.Collapsed || a.chat.details.cancel != nil {
		return
	}
	id, ref := layout.PendingDetail()
	if ref == nil {
		return
	}
	store, ok := a.opts.Store.(interface {
		ToolEvent(context.Context, runrecord.EventReference, history.ReadBounds) (runrecord.Event, error)
	})
	if !ok {
		return
	}
	ctx, cancel := context.WithCancel(a.ctx)
	a.chat.details = detailRead{cancel: cancel}
	result := detailResult{id: id, revision: a.chat.historyRevision, reference: *ref}
	language, redact := a.language, a.opts.Vault.Redact
	a.workers.Add(1)
	go func() {
		defer a.workers.Done()
		defer cancel()
		event, err := store.ToolEvent(ctx, result.reference, history.ReadBounds{})
		if err == nil {
			err = ctx.Err()
		}
		result.err = err
		if err == nil {
			event.Reference = nil
			event.Agent = ""
			blocks := (transcript.Formatter{Language: language, Redact: redact}).Event(event)
			if len(blocks) > 0 {
				result.block = blocks[0]
			}
		}
		if err := ctx.Err(); err != nil {
			result.err, result.block = err, viewport.Block{}
		}
		a.post(a.ctx, notice{Detail: &result})
	}()
}
func (a *app) receiveToolDetail(result *detailResult) {
	read := a.chat.details
	a.chat.details = detailRead{}
	if read.cancel == nil || read.abandoned || result.revision != a.chat.historyRevision || a.chat.output.Collapsed {
		return
	}
	if result.err != nil {
		result.block.Text = "\n  " + a.language.Text("Failed: ") + a.opts.Vault.Redact(result.err.Error()) + "\n"
	}
	a.chat.output.UpdateDetails(result.id, result.reference, result.block)
}

// Keep the slot until its worker responds so repeated toggles never overlap reads.
func (a *app) cancelToolDetail() {
	if read := &a.chat.details; read.cancel != nil && !read.abandoned {
		read.abandoned = true
		read.cancel()
	}
}
