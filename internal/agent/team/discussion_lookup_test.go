// Purpose: Reuse live identities for public feedback while validating historical names and cancellation.
// Pattern: Counted repository boundary test.
// Usage: go test -race ./internal/agent/team -run TestDiscussionKnown.
// Related: internal/agent/team/discussion.go, internal/agent/team/membership.go.
package team

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/team/identity"
)

type memberLookupRoom struct {
	*memoryRoom
	lookups atomic.Int64
}

func (s *memberLookupRoom) TeamMember(ctx context.Context, name string) (identity.Member, error) {
	s.lookups.Add(1)
	return s.memoryRoom.TeamMember(ctx, name)
}

func TestDiscussionKnownMembersAvoidRedundantQueriesAndRespectCancellation(t *testing.T) {
	r := testRoom(t)
	store := &memberLookupRoom{memoryRoom: r.options.Store.(*memoryRoom)}
	r.options.Store = store
	p := &Peer{ID: "Hana", room: r}
	if _, err := p.Discuss(r.ctx, "Consider the existing API.", 0, []string{"Ren"}); err != nil {
		t.Fatal(err)
	}
	if store.lookups.Load() != 0 {
		t.Fatal("live identity was queried again")
	}
	store.members = append(store.members, identity.Member{ID: "Arjun", Session: "session_closed"})
	if _, err := p.Discuss(r.ctx, "This may help a later review.", 0, []string{"Arjun", "Arjun"}); err != nil {
		t.Fatal(err)
	}
	if store.lookups.Load() != 1 || r.active["Arjun"] != nil {
		t.Fatal("historical identity was not validated or was restarted")
	}
	through := r.latestPublic
	ctx, cancel := context.WithCancel(r.ctx)
	cancel()
	if _, err := p.Discuss(ctx, "A cancelled opinion", 0, []string{"Ren"}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if r.latestPublic != through {
		t.Fatal("cancelled feedback was published")
	}
}
