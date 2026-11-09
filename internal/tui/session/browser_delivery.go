// Purpose: Translate completed browser work into ordered terminal notices.
// Pattern: Worker-to-event-loop adapter.
// Usage: Browser controller workers deliver immutable values through the application queue.
// Related: internal/tui/browser/controller/controller.go, internal/tui/session/events.go.
package session

import (
	"context"

	"github.com/keonho-kim/meca/internal/tui/browser/controller"
)

func (a *app) DeliverFiles(ctx context.Context, result controller.Result) error {
	n := notice{Directory: result.Directory, PreviewLayout: result.Layout, FileWatch: result.Watch}
	if result.Preview != nil {
		n.PreviewRevision = result.Preview.Request.Revision
		n.FilePreview = &result.Preview.Preview
		n.Err = result.Preview.Err
	}
	return a.post(ctx, n)
}
