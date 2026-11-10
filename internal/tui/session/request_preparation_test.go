// Purpose: Keep selected-path filesystem checks off keyboard handling and preserve their captured identity.
// Pattern: Channel-controlled request-worker test.
// Usage: go test -race ./internal/tui/session -run TestRequestPreparationRunsOnWorker.
// Related: internal/tui/session/tasks.go, internal/tui/composer/request.go.
package session

import (
	"context"
	"strings"
	"testing"
	"time"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	requestinput "github.com/keonho-kim/meca/internal/commands/input"
	"github.com/keonho-kim/meca/internal/fileview"
	"github.com/keonho-kim/meca/internal/policy"
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
	a.receiveRuntime(&runrecord.Event{Kind: "usage", Usage: &modelapi.TokenUsage{Output: 20}, MeasuredDuration: time.Second})
	a.startRequest(requestinput.Capture("review", []string{"captured.txt"}))
	if ctx.Err() != nil {
		t.Fatal("request preparation blocked the terminal event loop")
	}
	a.render()
	if line := screenLine(a.screen, 0, 80); strings.Contains(line, "초당 토큰") || !strings.HasSuffix(line, "출력 20") {
		t.Fatalf("new request reused old TPS or lost session totals: %q", line)
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
	a.compose.editor.Insert("input remains available")
	a.work.cancel()
	a.workers.Wait()
	if a.compose.editor.String() != "input remains available" {
		t.Fatal("background preparation changed the input")
	}
}
