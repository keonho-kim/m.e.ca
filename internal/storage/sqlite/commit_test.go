// Purpose: Verify grouped journal writes are atomic and retain ordered message/event history.
// Pattern: Persistence boundary tests.
// Usage: go test ./internal/storage/sqlite -run TestCommit.
// Related: internal/storage/sqlite/commit.go, internal/storage/sqlite/messages.go, internal/storage/sqlite/events.go.
package sqlite

import (
	"context"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

func TestCommitAtomicAndOrdered(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	id := s.Session().ID
	ctx := context.Background()
	m := modelapi.Message{Role: "user", Content: "grouped request"}
	e := runrecord.Event{Kind: "thinking", Thinking: "observed"}
	invalid := runrecord.Event{Kind: "usage"}
	if err := s.Commit(ctx, id, []runrecord.Entry{{Message: &m}, {Event: &invalid}}); err == nil {
		t.Fatal("invalid group committed")
	}
	if rows, _, err := s.Timeline(ctx, id, 0, 100); err != nil || len(rows) != 0 {
		t.Fatalf("partial group: %v %v", rows, err)
	}
	if err := s.Commit(ctx, id, []runrecord.Entry{{Message: &m}, {Event: &e}}); err != nil {
		t.Fatal(err)
	}
	rows, _, err := s.Timeline(ctx, id, 0, 100)
	if err != nil || len(rows) != 2 || rows[0].Kind != "message" || rows[1].Kind != "event" {
		t.Fatalf("order: %+v %v", rows, err)
	}
	ctx, cancel := context.WithCancel(ctx)
	cancel()
	if err := s.Commit(ctx, id, []runrecord.Entry{{Message: &m}}); err == nil {
		t.Fatal("cancelled commit succeeded")
	}
}
