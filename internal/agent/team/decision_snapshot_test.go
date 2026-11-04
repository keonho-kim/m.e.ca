// Purpose: Keep received public history and current work on one decision boundary during concurrent publication.
// Pattern: Controlled repository interleaving test.
// Usage: go test -race ./internal/agent/team -run TestDecisionSnapshot.
// Related: internal/agent/team/inbox.go, internal/agent/team/work_context.go.
package team

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/team/channel"
)

type snapshotRoom struct {
	*memoryRoom
	once             sync.Once
	entered, release chan struct{}
}

func (s *snapshotRoom) TeamMessages(ctx context.Context, after, through int64, limit int) ([]channel.Message, error) {
	s.once.Do(func() { close(s.entered); <-s.release })
	return s.memoryRoom.TeamMessages(ctx, after, through, limit)
}

func TestDecisionSnapshotDoesNotIncludeLaterWork(t *testing.T) {
	r := testRoom(t)
	hana, ren := &Peer{ID: "Hana", room: r}, &Peer{ID: "Ren", room: r}
	if _, err := hana.SetWork("Authentication", []string{"/project/auth"}, "old-contract", 0); err != nil {
		t.Fatal(err)
	}
	r.public = channel.History{}
	store := &snapshotRoom{memoryRoom: r.options.Store.(*memoryRoom), entered: make(chan struct{}), release: make(chan struct{})}
	r.options.Store = store
	done := make(chan error, 1)
	go func() { _, err := ren.Inbox(r.ctx); done <- err }()
	<-store.entered
	if _, err := hana.SetWork("Authentication", []string{"/project/auth"}, "new-contract", 0); err != nil {
		t.Fatal(err)
	}
	close(store.release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	ren.BeginDecision()
	if text := ren.ContextObservations(nil); !strings.Contains(text, "old-contract") || strings.Contains(text, "new-contract") {
		t.Fatal("a later work snapshot escaped its receive boundary", text)
	}
	if _, err := ren.Inbox(r.ctx); err != nil {
		t.Fatal(err)
	}
	ren.BeginDecision()
	if text := ren.ContextObservations(nil); !strings.Contains(text, "new-contract") {
		t.Fatal("new work did not reach the next decision", text)
	}
}
