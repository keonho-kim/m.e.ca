// Purpose: Keep private-session preparation independent of public conversation admission.
// Pattern: Controlled repository interleaving tests.
// Usage: Team tests block creation while inspecting publication and steering behavior.
// Related: internal/agent/team/membership.go, internal/agent/team/decision.go.
package team

import (
	"context"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/team/channel"
	"github.com/keonho-kim/meca/internal/agent/team/identity"
)

type slowRegistration struct {
	*memoryRoom
	entered, release chan struct{}
}

func TestInstructionDuringSessionCreationPreventsLaunchAndDiscardsNewState(t *testing.T) {
	r := testRoom(t)
	store := &slowRegistration{memoryRoom: r.options.Store.(*memoryRoom), entered: make(chan struct{}), release: make(chan struct{})}
	r.options.Store = store
	p := &Peer{ID: "Hana", room: r}
	done := make(chan error, 1)
	go func() { _, err := p.AddMember(r.ctx, "", "Change authentication"); done <- err }()
	<-store.entered
	if _, err := r.publish(channel.Message{From: "user", Text: "Do not add another teammate"}); err != nil {
		close(store.release)
		<-done
		t.Fatal(err)
	}
	close(store.release)
	if err := <-done; err == nil {
		t.Fatal("obsolete invitation launched a peer")
	}
	members, err := store.TeamMembers(r.ctx, "", 100)
	if err != nil || len(members) != 2 || r.countLocked() != 2 {
		t.Fatal("obsolete invitation retained unused session state", members, err)
	}
}

func (s *slowRegistration) CreateTeamAgent(ctx context.Context) (identity.Member, error) {
	close(s.entered)
	select {
	case <-s.release:
	case <-ctx.Done():
		return identity.Member{}, ctx.Err()
	}
	return s.memoryRoom.CreateTeamAgent(ctx)
}

func TestSessionCreationLeavesPublicJournalAvailable(t *testing.T) {
	r := testRoom(t)
	store := &slowRegistration{memoryRoom: r.options.Store.(*memoryRoom), entered: make(chan struct{}), release: make(chan struct{})}
	r.options.Store = store
	p := &Peer{ID: "Hana", room: r}
	done := make(chan error, 1)
	go func() { _, err := p.AddMember(r.ctx, "", "Inspect an independent file"); done <- err }()
	<-store.entered
	available := r.logMu.TryLock()
	if available {
		r.logMu.Unlock()
	}
	close(store.release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if !available {
		t.Fatal("private-session preparation blocked public messages and user input")
	}
}
