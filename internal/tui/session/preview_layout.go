// Purpose: Schedule only one preview calculation and its latest replacement without blocking input.
// Pattern: Revision-bound single-worker bulkhead.
// Usage: File rendering requests geometry; notices transfer completed layouts to the event loop.
// Related: internal/tui/preview/build.go, internal/tui/browser/readqueue/slot.go, internal/tui/session/render_files.go.
package session

import (
	"github.com/keonho-kim/meca/internal/tui/browser/controller"

	"github.com/keonho-kim/meca/internal/tui/preview"
)

func (a *app) ensurePreviewLayout(width int) {
	p := a.files.filePreview
	if p == nil || !p.NeedsLayout(width) {
		return
	}
	source := p.Snapshot(width)
	if a.files.reads.Layout.Running() && a.files.previewLayoutDesired == source {
		return
	}
	p.ReleaseLayout()
	a.files.previewLayoutDesired = source
	a.files.previewLayoutRevision++
	request := controller.LayoutRequest{Source: source, Revision: a.files.previewLayoutRevision}
	if a.files.reads.Layout.Submit(request) {
		a.startPreviewLayout(request)
	}
}

func (a *app) startPreviewLayout(request controller.LayoutRequest) {
	a.files.reads.BuildLayout(a.ctx, a, request)
}

func (a *app) receivePreviewLayout(result *controller.LayoutResult) {
	if next := a.files.reads.Layout.Complete(result.Request.Revision); next != nil && !a.exit && a.ctx.Err() == nil {
		a.startPreviewLayout(*next)
	}
	if result.Request.Revision != a.files.previewLayoutRevision || a.files.filePreview == nil || result.Err != nil {
		return
	}
	a.files.filePreview.Apply(result.Request.Source, result.Layout)
}

func (a *app) stopPreviewLayout() {
	a.files.reads.Layout.Stop()
	a.files.previewLayoutRevision++
	a.files.previewLayoutDesired = preview.Snapshot{}
}
