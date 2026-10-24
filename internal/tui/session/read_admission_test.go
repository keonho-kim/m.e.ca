// Purpose: Verify rapid file navigation runs only the active read and the latest replacement.
// Pattern: Channel-controlled I/O boundary tests.
// Usage: go test -race ./internal/tui/session -run TestReadAdmission.
// Related: internal/tui/session/directory.go, internal/tui/session/file_preview.go, internal/tui/session/tools.go.
package session

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/keonho-kim/meca/internal/policy"
	"github.com/keonho-kim/meca/internal/tools/local"
)

type readAttempt struct {
	path string
	ctx  context.Context
}

type heldReader struct {
	Tools
	requests chan readAttempt
	release  chan struct{}
	once     sync.Once
}

func (r *heldReader) unblock() { r.once.Do(func() { close(r.release) }) }

func (r *heldReader) read(ctx context.Context, path string) {
	r.requests <- readAttempt{path, ctx}
	// Simulate an OS read that cannot finish merely because its context was cancelled.
	<-r.release
}

func (r *heldReader) Reference(_ policy.Settings, path string) (local.Reference, error) {
	return local.Reference{Path: path, Directory: true}, nil
}

func (r *heldReader) Directory(ctx context.Context, _ policy.Settings, path string, _, _ int) ([]local.Entry, bool, error) {
	r.read(ctx, path)
	return nil, false, nil
}

func (r *heldReader) Preview(ctx context.Context, _ policy.Settings, path string) (local.FilePreview, error) {
	r.read(ctx, path)
	return local.FilePreview{Path: path, Content: path}, nil
}

func controlledReader(t *testing.T, a *app) *heldReader {
	t.Helper()
	r := &heldReader{Tools: a.opts.Tools, requests: make(chan readAttempt, 64), release: make(chan struct{})}
	a.opts.Tools = r
	t.Cleanup(r.unblock)
	return r
}

func firstRead(t *testing.T, r *heldReader) readAttempt {
	t.Helper()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	select {
	case attempt := <-r.requests:
		return attempt
	case <-timer.C:
		t.Fatal("read worker did not start")
		return readAttempt{}
	}
}

func TestReadAdmissionDirectoryKeepsLatest(t *testing.T) {
	a := fixture(t)
	r := controlledReader(t, a)
	a.page = "files"
	a.requestDirectory("open", "first", 0)
	first := firstRead(t, r)
	a.requestDirectory("open", "discarded", 0)
	a.requestDirectory("open", "latest", 0)
	if first.ctx.Err() == nil {
		t.Fatal("replacement did not cancel the active read")
	}
	r.unblock()
	finishBrowserIO(t, a)
	a.workers.Wait()
	if len(r.requests) != 1 || (<-r.requests).path != "latest" {
		t.Fatal("directory navigation ran obsolete queued reads")
	}
	if a.browser.Current() == nil || a.browser.Current().Path != "latest" {
		t.Fatal("latest directory did not become visible")
	}
}

func TestReadAdmissionPreviewKeepsLatest(t *testing.T) {
	a := fixture(t)
	r := controlledReader(t, a)
	a.page = "files"
	a.browser.Open(".", []local.Entry{{Name: "first"}, {Name: "discarded"}, {Name: "latest"}}, false)
	a.openFilePreview(false)
	firstRead(t, r)
	for _, index := range []int{1, 2} {
		a.browser.Current().Index = index
		a.openFilePreview(true)
	}
	r.unblock()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	for a.filePreview.Loading {
		select {
		case n := <-a.events:
			a.receive(n)
		case <-timer.C:
			t.Fatal("latest preview did not finish")
		}
	}
	a.workers.Wait()
	if len(r.requests) != 1 || !strings.HasSuffix((<-r.requests).path, "latest") || a.filePreview.Content != "latest" {
		t.Fatal("preview navigation ran obsolete queued reads")
	}
}

func TestReadAdmissionCloseDropsPending(t *testing.T) {
	a := fixture(t)
	r := controlledReader(t, a)
	a.page = "files"
	a.browser.Open(".", []local.Entry{{Name: "first"}, {Name: "discarded"}}, false)
	a.openFilePreview(false)
	firstRead(t, r)
	a.browser.Current().Index = 1
	a.openFilePreview(false)
	a.closeFilePreview()
	r.unblock()
	a.workers.Wait()
	for len(a.events) > 0 {
		a.receive(<-a.events)
	}
	if len(r.requests) != 0 || a.filePreview != nil {
		t.Fatal("closing preview retained or ran a pending read")
	}
}
