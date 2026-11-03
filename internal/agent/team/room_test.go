// Purpose: Verify autonomous session lifecycle, bounded invitations and restoration without real inference.
// Pattern: Deterministic concurrent application tests.
// Usage: go test -race ./internal/agent/team.
// Related: internal/agent/team/room.go, internal/agent/team/membership.go.
package team

import (
	"context"
	"crypto/rand"
	"fmt"
	"github.com/keonho-kim/meca/internal/agent/team/channel"
	"github.com/keonho-kim/meca/internal/agent/team/identity"
	"sync"
	"testing"
	"time"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/run"
)

type memoryRoom struct {
	mu       sync.Mutex
	members  []identity.Member
	messages []channel.Message
}

func (s *memoryRoom) TeamMembers(_ context.Context, after string, limit int) ([]identity.Member, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	start := 0
	for i, m := range s.members {
		if m.ID == after {
			start = i + 1
			break
		}
	}
	return append([]identity.Member(nil), s.members[start:min(start+limit, len(s.members))]...), nil
}
func (s *memoryRoom) TeamMember(_ context.Context, id string) (identity.Member, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, m := range s.members {
		if m.ID == id {
			return m, nil
		}
	}
	return identity.Member{}, fmt.Errorf("unknown peer")
}
func (s *memoryRoom) SetTeamSummary(_ context.Context, id, summary string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, m := range s.members {
		if m.ID == id {
			s.members[i].Summary = summary
			return nil
		}
	}
	return fmt.Errorf("unknown peer")
}
func (s *memoryRoom) CreateTeamAgent(context.Context) (identity.Member, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	m := identity.Member{ID: identity.Name(len(s.members)), Session: "session_" + rand.Text()}
	s.members = append(s.members, m)
	return m, nil
}
func (s *memoryRoom) TeamPending(_ context.Context, after, through int64, own string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, m := range s.messages {
		if m.Sequence > after && m.Sequence <= through && (m.From != own || !m.FromStream) {
			return true, nil
		}
	}
	return false, nil
}
func (s *memoryRoom) AppendTeamMessage(_ context.Context, m channel.Message) (channel.Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	m.Sequence = int64(len(s.messages) + 1)
	s.messages = append(s.messages, m)
	return m, nil
}
func (s *memoryRoom) TeamMessages(_ context.Context, after, through int64, limit int) ([]channel.Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if through == 0 {
		through = int64(len(s.messages))
	}
	end := min(int(after)+limit, len(s.messages), int(through))
	if after >= int64(end) {
		return nil, nil
	}
	return append([]channel.Message(nil), s.messages[after:end]...), nil
}
func quiet(runrecord.Event) error { return nil }

func TestRoomRestoresInitialPairAndInvitedPeer(t *testing.T) {
	store := &memoryRoom{}
	var first []identity.Member
	for request := range 2 {
		invited := make(chan struct{})
		departed := make(chan struct{}, 2)
		execute := func(ctx context.Context, member identity.Member, _ string, peer *Peer) (run.Result, error) {
			if _, err := peer.Inbox(ctx); err != nil {
				return run.Result{}, err
			}
			if member.ID == "agent-a" {
				id := ""
				if request > 0 {
					id = "agent-c"
				}
				if _, err := peer.Invite(ctx, id, "Check the implementation"); err != nil {
					return run.Result{}, err
				}
				close(invited)
			}
			if member.ID == "agent-b" {
				<-invited
			}
			if member.ID != "agent-c" {
				leave, err := peer.Goodbye(ctx, "Contribution finished")
				departed <- struct{}{}
				return run.Result{Departed: leave}, err
			}
			<-departed
			<-departed
			if _, err := peer.Inbox(ctx); err != nil {
				return run.Result{}, err
			}
			return run.Result{Text: "Implemented and checked"}, nil
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		result, err := Run(ctx, Options{Store: store, Execute: execute, Emit: quiet, MaxModelCalls: 100, MaxSeconds: 5}, "Build the requested feature")
		cancel()
		if err != nil || result.Agent != "agent-c" || result.Text == "" {
			t.Fatalf("result=%+v err=%v", result, err)
		}
		members, _ := store.TeamMembers(context.Background(), "", 100)
		if len(members) != 3 {
			t.Fatalf("unexpected members %+v", members)
		}
		if request == 0 {
			first = members
		} else {
			for i := range first {
				if first[i].Session != members[i].Session {
					t.Fatal("restoration replaced a session")
				}
			}
		}
	}
}

func testRoom(t *testing.T) *Room {
	t.Helper()
	ctx, cancel := context.WithCancelCause(context.Background())
	r := &Room{ctx: ctx, cancel: cancel, active: map[string]*participant{}, changed: make(chan struct{}, 1), deadline: time.Now().Add(time.Minute)}
	r.options = Options{Store: &memoryRoom{}, Emit: quiet, Execute: func(ctx context.Context, _ identity.Member, _ string, _ *Peer) (run.Result, error) {
		<-ctx.Done()
		return run.Result{}, ctx.Err()
	}}
	for i := range 2 {
		member := identity.Member{ID: identity.Name(i), Session: "session_" + rand.Text()}
		r.options.Store.(*memoryRoom).members = append(r.options.Store.(*memoryRoom).members, member)
		r.active[member.ID] = &participant{member: member, wake: make(chan struct{}, 1)}
	}
	t.Cleanup(func() { cancel(nil); r.workers.Wait() })
	return r
}
func TestConcurrentInvitationsReserveOnlyEightSlots(t *testing.T) {
	r := testRoom(t)
	p := &Peer{ID: "agent-a", room: r}
	var workers sync.WaitGroup
	for range 20 {
		workers.Go(func() {
			if _, err := p.Invite(r.ctx, "", "Help independently"); err != nil {
				t.Error(err)
			}
		})
	}
	workers.Wait()
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.countLocked() != 8 || len(r.options.Store.(*memoryRoom).members) != 8 {
		t.Fatalf("active=%d members=%d", r.countLocked(), len(r.options.Store.(*memoryRoom).members))
	}
}
func TestConcurrentGoodbyesLeaveExactlyOneReporter(t *testing.T) {
	r := testRoom(t)
	values := make(chan bool, 2)
	var workers sync.WaitGroup
	for _, id := range []string{"agent-a", "agent-b"} {
		workers.Go(func() {
			left, err := (&Peer{ID: id, room: r}).Goodbye(r.ctx, "Done")
			if err != nil {
				t.Error(err)
			}
			values <- left
		})
	}
	workers.Wait()
	close(values)
	departures := 0
	for left := range values {
		if left {
			departures++
		}
	}
	if departures != 1 {
		t.Fatalf("departures=%d", departures)
	}
}
func TestPublicProgressReachesEveryPeer(t *testing.T) {
	r := testRoom(t)
	a := &Peer{ID: "agent-a", room: r}
	if err := a.Emit(runrecord.Event{Kind: "progress", Text: "I will update api/handler.go using the existing interface."}); err != nil {
		t.Fatal(err)
	}
	c := &Peer{ID: "agent-c", room: r}
	batch, err := c.Inbox(r.ctx)
	if err != nil || len(batch.Messages) != 1 {
		t.Fatalf("batch=%+v error=%v", batch, err)
	}
	if next, err := c.Inbox(r.ctx); err != nil || len(next.Messages) != 0 {
		t.Fatal(fmt.Sprint(next, err))
	}
}
