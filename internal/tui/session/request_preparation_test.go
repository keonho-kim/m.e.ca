// Purpose: Keep selected-path filesystem checks off keyboard handling and preserve their captured identity.
// Pattern: Channel-controlled request-worker test.
// Usage: go test -race ./internal/tui/session -run TestRequestPreparationRunsOnWorker.
// Related: internal/tui/session/tasks.go, internal/tui/composer/request.go.
package session

import (
	"context"
	"testing"
	"time"

	"github.com/keonho-kim/meca/internal/fileview"
	"github.com/keonho-kim/meca/internal/policy"
	"github.com/keonho-kim/meca/internal/tui/composer"
)

type blockedReference struct {
	fileview.Source
	started chan string
}

func (r *blockedReference) Reference(ctx context.Context, _ policy.Settings, path string) (fileview.Reference, error) {
	r.started <- path
	<-ctx.Done()
	return fileview.Reference{}, ctx.Err()
}

func TestRequestPreparationRunsOnWorker(t *testing.T) {
	a := fixture(t)
	reader := &blockedReference{Source: a.opts.Files, started: make(chan string, 1)}
	a.opts.Files = reader
	ctx, cancel := context.WithTimeout(a.ctx, 5*time.Second)
	a.ctx = ctx
	defer cancel()
	a.startRequest(composer.Capture("review", []string{"captured.txt"}))
	if ctx.Err() != nil {
		t.Fatal("request preparation blocked the terminal event loop")
	}
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	select {
	case path := <-reader.started:
		if path != "captured.txt" {
			t.Fatal(path)
		}
	case <-timer.C:
		t.Fatal("request preparation did not start")
	}
	a.editor.Insert("input remains available")
	a.taskCancel()
	a.workers.Wait()
	if a.editor.String() != "input remains available" {
		t.Fatal("background preparation changed the input")
	}
}
