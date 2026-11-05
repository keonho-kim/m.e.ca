// Purpose: Discard canceled detail reads and resume one fresh read after rapid folding.
// Pattern: Controlled cancellable reader and virtual-terminal regression tests.
// Usage: go test -race ./internal/tui/session -run TestDetailRead.
// Related: internal/tui/session/tool_details.go, internal/tui/session/keys.go.
package session

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/tui/viewport"
)

type cancelableDetailStore struct {
	Store
	started chan struct{}
	calls   atomic.Int32
}

func (s *cancelableDetailStore) ToolEvent(ctx context.Context, _ runrecord.EventReference) (runrecord.Event, error) {
	if s.calls.Add(1) == 1 {
		close(s.started)
		<-ctx.Done()
		return runrecord.Event{}, ctx.Err()
	}
	return runrecord.Event{Kind: "tool_succeeded", Call: &toolapi.Call{Name: "execute_command", Arguments: []byte(`{"command":"pwd"}`)}, Output: []byte(`{"Stdout":"actual output","Status":"exited","ExitCode":0}`)}, nil
}

func TestDetailReadCanceledBeforeReopenIsDiscarded(t *testing.T) {
	a := screenFixture(t)
	a.chat.output.Collapsed = false
	store := &cancelableDetailStore{started: make(chan struct{})}
	a.opts.Store = store
	a.chat.output.AppendBlock(viewport.Block{Text: "Command summary", Summary: "Command summary", DetailRef: &runrecord.EventReference{Session: "private", Sequence: 1}})
	a.render()
	select {
	case <-store.started:
	case <-time.After(5 * time.Second):
		t.Fatal("detail read did not begin")
	}
	a.toggleToolDetails()
	a.toggleToolDetails()
	select {
	case event := <-a.events:
		a.receive(event)
	case <-time.After(5 * time.Second):
		t.Fatal("canceled detail read did not return")
	}
	block := a.chat.output.Blocks[0]
	if strings.Contains(block.Text, "context canceled") || block.Details != "" {
		t.Fatalf("reopened result adopted a canceled detail: %q", block.Text)
	}
	settleVisibleDetails(t, a)
	if store.calls.Load() != 2 || !strings.Contains(a.chat.output.Blocks[0].Text, "actual output") {
		t.Fatal("reopening did not load one fresh result")
	}
}
