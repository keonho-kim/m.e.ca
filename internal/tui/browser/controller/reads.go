// Purpose: Run one directory, preview or layout computation per existing read slot.
// Pattern: Event-loop-owned lifecycle with joined workers.
// Usage: The terminal session submits reads and accepts immutable results.
// Related: internal/tui/session/directory.go, internal/tui/session/file_preview.go, internal/tui/browser/readqueue/slot.go.
package controller

import (
	"context"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/fileview"
	"github.com/keonho-kim/meca/internal/policy"
	"github.com/keonho-kim/meca/internal/tui/preview"
)

func (c *Controller) ReadDirectory(parent context.Context, sink Sink, reader Reader, settings policy.Settings, page DirectoryPage) {
	owner := c.begin(parent, sink)
	if owner == nil {
		return
	}
	ctx, cancel := context.WithCancel(owner)
	c.Directory.Begin(page.Revision, cancel)
	c.workers.Go(func() {
		defer cancel()
		if page.Mode == "open" {
			ref, err := reader.Reference(ctx, settings, page.Path)
			page.Err = err
			if err == nil && !ref.Directory {
				page.Err = faults.Errorf(diagcatalog.ControllerNotADirectory, page.Path)
			}
			page.Path = ref.Path
		}
		if page.Err == nil && ctx.Err() == nil {
			value, err := reader.Directory(ctx, settings, page.Path, page.Offset, 1000, "")
			page.Entries, page.More, page.Err = value.Entries, value.More, err
		}
		if ctx.Err() != nil {
			page.Entries, page.Err = nil, ctx.Err()
		}
		c.sink.DeliverFiles(owner, Result{Directory: &page})
	})
}
func (c *Controller) ReadPreview(parent context.Context, sink Sink, reader Reader, settings policy.Settings, request PreviewRequest) {
	owner := c.begin(parent, sink)
	if owner == nil {
		return
	}
	ctx, cancel := context.WithCancel(owner)
	c.Preview.Begin(request.Revision, cancel)
	c.workers.Go(func() {
		defer cancel()
		value, err := reader.Preview(ctx, settings, request.Path)
		if ctx.Err() != nil {
			value = fileview.Preview{}
			err = ctx.Err()
		}
		c.sink.DeliverFiles(owner, Result{Preview: &PreviewResult{Request: request, Preview: value, Err: err}})
	})
}
func (c *Controller) BuildLayout(parent context.Context, sink Sink, request LayoutRequest) {
	owner := c.begin(parent, sink)
	if owner == nil {
		return
	}
	ctx, cancel := context.WithCancel(owner)
	c.Layout.Begin(request.Revision, cancel)
	c.workers.Go(func() {
		defer cancel()
		layout, err := preview.Build(ctx, request.Source)
		c.sink.DeliverFiles(owner, Result{Layout: &LayoutResult{Request: request, Layout: layout, Err: err}})
	})
}
