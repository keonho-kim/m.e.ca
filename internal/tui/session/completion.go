// Purpose: Own cancellable completion searches and reject stale candidate results.
// Pattern: Presentation worker lifecycle.
// Usage: Called after editing the current reference token in the terminal loop.
// Related: internal/tui/input/editor.go, internal/tools/local/references.go.
package session

import (
	"context"
)

func (a *app) closeReferences() {
	if a.refCancel != nil {
		a.refCancel()
		a.refCancel = nil
	}
	a.refOpen = false
	a.refs = nil
	a.refRevision++
}
func (a *app) refreshReferences() {
	token, start, ok := a.editor.Reference()
	if !ok {
		a.closeReferences()
		return
	}
	if a.refCancel != nil {
		a.refCancel()
	}
	ctx, cancel := context.WithCancel(a.ctx)
	a.refCancel = cancel
	a.refRevision++
	revision := a.refRevision
	a.refOpen = true
	a.refStart = start
	a.refs = nil
	a.refIndex = 0
	settings := a.settings
	a.workers.Add(1)
	go func() {
		defer a.workers.Done()
		defer cancel()
		refs, err := a.opts.Tools.References(ctx, settings, token)
		if ctx.Err() == nil {
			a.post(ctx, notice{Revision: revision, References: refs, Err: err})
		}
	}()
}
