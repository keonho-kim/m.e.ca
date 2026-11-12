// Purpose: Require explicit original-read admission and release it after terminal adoption.
// Pattern: Controlled tool-reader resource contract test.
// Usage: go test ./internal/tui/session -run TestToolDetailMemoryBudget.
// Related: internal/tui/session/tool_details.go, internal/processmemory/payload.go.
package session

import (
	"context"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/processmemory"
	"github.com/keonho-kim/meca/internal/storage/history"
	"github.com/keonho-kim/meca/internal/tui/viewport"
)

type boundedDetailStore struct {
	Store
	bounds chan history.ReadBounds
}

func (s *boundedDetailStore) ToolEvent(_ context.Context, _ record.EventReference, bounds history.ReadBounds) (record.Event, error) {
	s.bounds <- bounds
	if bounds.Reserve != nil {
		if err := bounds.Reserve(1024); err != nil {
			return record.Event{}, err
		}
	}
	return record.Event{Kind: "tool_succeeded", Call: &tool.Call{Name: "execute_command", Arguments: []byte(`{"command":"pwd"}`)}, Output: []byte(`{"Status":"exited","ExitCode":0,"Stdout":"safe output"}`)}, nil
}

func TestToolDetailMemoryBudgetIsOwnedThroughAdoption(t *testing.T) {
	a := screenFixture(t)
	store := &boundedDetailStore{bounds: make(chan history.ReadBounds, 1)}
	a.opts.Store = store
	a.chat.output.Collapsed = false
	a.chat.output.AppendBlock(viewport.Block{Text: "tool summary", Summary: "tool summary", DetailRef: &record.EventReference{Session: "session", Sequence: 1}})
	settleVisibleDetails(t, a)
	bounds := <-store.bounds
	if bounds.MaxBytes <= 0 || bounds.Reserve == nil {
		t.Fatal("terminal original read has no explicit byte or admission bound")
	}
	lease := processmemory.NewPayloadLease()
	defer lease.Close()
	if err := lease.Resize(int(processmemory.PayloadBytes)); err != nil {
		t.Fatal("detail adoption retained its transient reservation", err)
	}
}
