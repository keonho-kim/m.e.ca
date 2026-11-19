// Purpose: Release private observations and expose unread cursors for the active run.
// Pattern: Lifecycle composition over the browser tool session.
// Usage: The agent restores cursors after compaction and clears them at completion.
// Related: internal/tools/local/browser.go, internal/tools/local/webfetch.go.
package local

import (
	"context"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func (e *Executor) EndObservations(ctx context.Context) error {
	if e.browser == nil {
		return nil
	}
	return e.browser.Close(ctx)
}

func (e *Executor) ObservationCursors(ctx context.Context) ([]toolapi.ObservationCursor, error) {
	if e.browser == nil {
		return nil, nil
	}
	return e.browser.Cursors(ctx)
}
