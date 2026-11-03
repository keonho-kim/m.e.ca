// Purpose: Own browser worker lifetimes and bounded per-feature read slots.
// Pattern: Event-loop-owned lifecycle with joined workers.
// Usage: The terminal session submits reads and accepts immutable results.
// Related: internal/tui/session/directory.go, internal/tui/session/file_preview.go, internal/tui/browser/readqueue/slot.go.
package controller

import (
	"context"
	"github.com/keonho-kim/meca/internal/fileview"
	"github.com/keonho-kim/meca/internal/policy"
	"github.com/keonho-kim/meca/internal/tui/browser/readqueue"
	"github.com/keonho-kim/meca/internal/tui/preview"
	"sync"
)

type Reader interface {
	Reference(context.Context, policy.Settings, string) (fileview.Reference, error)
	Directory(context.Context, policy.Settings, string, int, int) ([]fileview.Entry, bool, error)
	Preview(context.Context, policy.Settings, string) (fileview.Preview, error)
}
type Sink interface {
	DeliverFiles(context.Context, Result) error
}
type Result struct {
	Directory *DirectoryPage
	Preview   *PreviewResult
	Layout    *LayoutResult
	Watch     *WatchResult
}

// Slots and Watch are controlled only by the terminal event loop. Workers return
// immutable results through Sink and never mutate the UI or the slot state.
type Controller struct {
	Directory     readqueue.Slot[DirectoryPage]
	Preview       readqueue.Slot[PreviewRequest]
	Layout        readqueue.Slot[LayoutRequest]
	WatchReads    readqueue.Slot[WatchRequest]
	WatchRevision int
	Watch         Watch
	ctx           context.Context
	cancel        context.CancelFunc
	workers       sync.WaitGroup
	sink          Sink
	closed        bool
}

func (c *Controller) begin(ctx context.Context, sink Sink) context.Context {
	if c.closed {
		return nil
	}
	if c.ctx == nil {
		c.ctx, c.cancel = context.WithCancel(ctx)
		c.sink = sink
	}
	return c.ctx
}
func (c *Controller) Wait() { c.workers.Wait() }
func (c *Controller) Close() error {
	if c.closed {
		return nil
	}
	c.closed = true
	if c.cancel != nil {
		c.cancel()
	}
	c.Directory.Stop()
	c.Preview.Stop()
	c.Layout.Stop()
	err := c.StopWatch()
	c.workers.Wait()
	c.Directory = readqueue.Slot[DirectoryPage]{}
	c.Preview = readqueue.Slot[PreviewRequest]{}
	c.Layout = readqueue.Slot[LayoutRequest]{}
	c.WatchReads = readqueue.Slot[WatchRequest]{}
	c.sink = nil
	c.ctx, c.cancel = nil, nil
	return err
}

type DirectoryPage struct {
	Revision, Offset int
	Mode, Path       string
	Entries          []fileview.Entry
	More             bool
	Err              error
}
type PreviewRequest struct {
	Path     string
	Revision int
}
type PreviewResult struct {
	Request PreviewRequest
	Preview fileview.Preview
	Err     error
}
type LayoutRequest struct {
	Source   preview.Snapshot
	Revision int
}
type LayoutResult struct {
	Request LayoutRequest
	Layout  *preview.Layout
	Err     error
}
