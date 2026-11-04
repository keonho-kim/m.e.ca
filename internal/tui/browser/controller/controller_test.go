// Purpose: Verify browser cancellation retains active slots and shutdown joins delivery before release.
// Pattern: Deterministic resource lifecycle tests.
// Usage: go test -race ./internal/tui/browser/controller.
// Related: internal/tui/browser/controller/controller.go, internal/tui/browser/controller/reads.go.
package controller

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/keonho-kim/meca/internal/fileview"
	"github.com/keonho-kim/meca/internal/policy"
)

type previewReader struct {
	Reader
	entered, release chan struct{}
	calls            atomic.Int64
}

func (r *previewReader) Preview(ctx context.Context, _ policy.Settings, path string) (fileview.Preview, error) {
	r.calls.Add(1)
	if r.entered != nil {
		close(r.entered)
		<-r.release
	}
	return fileview.Preview{Path: path, Content: "current content"}, ctx.Err()
}

type resultSink struct{ values chan Result }

func (s resultSink) DeliverFiles(ctx context.Context, result Result) error {
	select {
	case s.values <- result:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func TestCancelledReadKeepsSlotUntilActualReturn(t *testing.T) {
	var c Controller
	reader := &previewReader{entered: make(chan struct{}), release: make(chan struct{})}
	release := sync.OnceFunc(func() { close(reader.release) })
	t.Cleanup(func() { release(); c.Close() })
	sink := resultSink{values: make(chan Result, 1)}
	c.ReadPreview(context.Background(), sink, reader, policy.Settings{}, PreviewRequest{Path: "first", Revision: 1})
	<-reader.entered
	for _, revision := range []int{2, 3} {
		if c.Preview.Submit(PreviewRequest{Path: "latest", Revision: revision}) {
			t.Fatal("cancelled I/O released its physical slot before returning")
		}
	}
	release()
	result := <-sink.values
	c.Wait()
	if !errors.Is(result.Preview.Err, context.Canceled) || result.Preview.Preview.Content != "" {
		t.Fatal("cancelled source remained usable")
	}
	if next := c.Preview.Complete(1); next == nil || next.Revision != 3 || c.Preview.Running() {
		t.Fatal("completion lost the latest pending read")
	}
}

type blockedSink struct{ entered, joined chan struct{} }

func (s blockedSink) DeliverFiles(ctx context.Context, _ Result) error {
	close(s.entered)
	<-ctx.Done()
	close(s.joined)
	return ctx.Err()
}

func TestCloseCancelsAndJoinsBlockedDelivery(t *testing.T) {
	var c Controller
	reader := &previewReader{}
	sink := blockedSink{entered: make(chan struct{}), joined: make(chan struct{})}
	c.ReadPreview(context.Background(), sink, reader, policy.Settings{}, PreviewRequest{Path: "file", Revision: 1})
	<-sink.entered
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-sink.joined:
	default:
		t.Fatal("close returned before its delivery worker")
	}
	if c.Preview.Running() || c.sink != nil || c.ctx != nil {
		t.Fatal("closed controller retained execution references")
	}
	c.ReadPreview(context.Background(), sink, reader, policy.Settings{}, PreviewRequest{Path: "again", Revision: 2})
	if reader.calls.Load() != 1 {
		t.Fatal("closed controller restarted I/O")
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
}
