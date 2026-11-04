// Purpose: Coalesce concurrent public feedback without dropping observations or repeating model attention.
// Pattern: Deterministic concurrent room test.
// Usage: go test -race ./internal/agent/team -run TestConcurrentDiscussion.
// Related: internal/agent/team/discussion.go, internal/agent/team/inbox.go.
package team

import (
	"fmt"
	"sync"
	"testing"
)

func TestConcurrentDiscussionCoalescesAttentionWithoutDroppingPublicNotes(t *testing.T) {
	r := testRoom(t)
	hana, ren := &Peer{ID: "Hana", room: r}, &Peer{ID: "Ren", room: r}
	work, err := hana.SetWork("Authentication", []string{"/project/auth"}, "Use 423", 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []*Peer{hana, ren} {
		if _, err := p.Inbox(r.ctx); err != nil {
			t.Fatal(err)
		}
		p.BeginDecision()
	}
	var senders sync.WaitGroup
	for i := range 8 {
		senders.Go(func() {
			if _, err := ren.Discuss(r.ctx, fmt.Sprintf("Independent finding %d", i), work.Sequence, nil); err != nil {
				t.Error(err)
			}
		})
	}
	senders.Wait()
	if len(r.active["Hana"].wake) != 1 {
		t.Fatal("notifications were not coalesced")
	}
	batch, err := hana.Inbox(r.ctx)
	if err != nil || len(batch.Messages) != 8 || !batch.Reconsider {
		t.Fatal("concurrent feedback was dropped", len(batch.Messages), batch, err)
	}
	hana.BeginDecision()
	if next, err := hana.Inbox(r.ctx); err != nil || next.Reconsider || len(next.Messages) != 0 {
		t.Fatal("consumed attention forced another inference", next, err)
	}
	if own, err := ren.Inbox(r.ctx); err != nil || len(own.Messages) != 0 || own.Reconsider {
		t.Fatal("native feedback returned to its author", own, err)
	}
}
