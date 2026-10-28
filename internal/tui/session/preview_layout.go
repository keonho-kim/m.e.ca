// Purpose: Schedule only one preview calculation and its latest replacement without blocking input.
// Pattern: Revision-bound single-worker bulkhead.
// Usage: File rendering requests geometry; notices transfer completed layouts to the event loop.
// Related: internal/tui/preview/build.go, internal/tui/browser/readqueue/slot.go, internal/tui/session/render_files.go.
package session

import (
	"context"

	"github.com/keonho-kim/meca/internal/tui/preview"
)

type previewLayoutRequest struct {
	source   preview.Snapshot
	revision int
}

type previewLayoutResult struct {
	request previewLayoutRequest
	layout  *preview.Layout
	err     error
}

func (a *app) ensurePreviewLayout(width int) {
	p := a.filePreview
	if p == nil || !p.NeedsLayout(width) {
		return
	}
	source := p.Snapshot(width)
	if a.previewLayouts.Running() && a.previewLayoutDesired == source {
		return
	}
	p.ReleaseLayout()
	a.previewLayoutDesired = source
	a.previewLayoutRevision++
	request := previewLayoutRequest{source: source, revision: a.previewLayoutRevision}
	if a.previewLayouts.Submit(request) {
		a.startPreviewLayout(request)
	}
}

func (a *app) startPreviewLayout(request previewLayoutRequest) {
	ctx, cancel := context.WithCancel(a.ctx)
	a.previewLayouts.Begin(request.revision, cancel)
	a.workers.Go(func() {
		defer cancel()
		layout, err := preview.Build(ctx, request.source)
		a.post(a.ctx, notice{PreviewLayout: &previewLayoutResult{request: request, layout: layout, err: err}})
	})
}

func (a *app) receivePreviewLayout(result *previewLayoutResult) {
	if next := a.previewLayouts.Complete(result.request.revision); next != nil && !a.exit && a.ctx.Err() == nil {
		a.startPreviewLayout(*next)
	}
	if result.request.revision != a.previewLayoutRevision || a.filePreview == nil || result.err != nil {
		return
	}
	a.filePreview.Apply(result.request.source, result.layout)
}

func (a *app) stopPreviewLayout() {
	a.previewLayouts.Stop()
	a.previewLayoutRevision++
	a.previewLayoutDesired = preview.Snapshot{}
}
