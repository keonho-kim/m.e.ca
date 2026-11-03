// Purpose: Verify stale wakeups never cause extra inference and active invitations never send another instruction.
// Pattern: Deterministic concurrency regression tests.
// Usage: go test -race ./internal/agent/team.
// Related: internal/agent/team/answer.go, internal/agent/team/membership.go.
package team

import (
	"context"
	"errors"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/team/channel"
	"strings"
	"testing"
)

type observedRoom struct {
	*memoryRoom
	checked chan struct{}
}

func (s observedRoom) TeamPending(ctx context.Context, after, through int64, own string) (bool, error) {
	pending, err := s.memoryRoom.TeamPending(ctx, after, through, own)
	s.checked <- struct{}{}
	return pending, err
}
func TestConsumedWakeKeepsCurrentExecutionIdle(t *testing.T) {
	r := testRoom(t)
	store := observedRoom{memoryRoom: r.options.Store.(*memoryRoom), checked: make(chan struct{}, 8)}
	r.options.Store = store
	peer := &Peer{ID: "Hana", room: r}
	if _, err := r.publish(channel.Message{From: "Ren", Text: "I will update api/router.go."}); err != nil {
		t.Fatal(err)
	}
	if _, err := peer.Inbox(r.ctx); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(r.ctx)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		again, err := peer.AfterAnswer(ctx, "I will update storage/query.go.")
		if again {
			err = errors.New("consumed wake resumed inference")
		}
		done <- err
	}()
	// The fully consumed prefix needs no SQL; only the newly published own answer is checked.
	<-store.checked
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
func TestActiveInvitationDoesNotPublishOrRestart(t *testing.T) {
	r := testRoom(t)
	peer := &Peer{ID: "Hana", room: r}
	result, err := peer.AddMember(r.ctx, "Ren", "Replace the active peer's work")
	if err != nil || result["status"] != "already_active" {
		t.Fatalf("%+v %v", result, err)
	}
	messages, err := peer.ReadMessages(r.ctx, 0)
	if err != nil || len(messages) != 0 || len(r.active["Ren"].wake) != 0 {
		t.Fatal("active peer received another instruction")
	}
}
func TestSlowDeliveryDoesNotHoldMembershipState(t *testing.T) {
	r := testRoom(t)
	entered, release := make(chan struct{}), make(chan struct{})
	r.options.Emit = func(event runrecord.Event) error { close(entered); <-release; return nil }
	done := make(chan error, 1)
	go func() { done <- (&Peer{ID: "Hana", room: r}).Emit(runrecord.Event{Kind: "stream", Text: "answer"}) }()
	<-entered
	if work := (&Peer{ID: "Ren", room: r}).WorkContext(); !strings.Contains(work, "Active teammates: Hana, Ren") {
		t.Fatal(work)
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestPrivateReasoningNeverEntersSharedDelivery(t *testing.T) {
	r := testRoom(t)
	var delivered []runrecord.Event
	r.options.Emit = func(event runrecord.Event) error { delivered = append(delivered, event); return nil }
	peer := &Peer{ID: "Hana", room: r}
	for _, event := range []runrecord.Event{{Kind: "thinking", Thinking: "private"}, {Kind: "stream", Thinking: "private"}, {Kind: "stream", Text: "I will update server/routes.go.", Thinking: "private"}} {
		if err := peer.Emit(event); err != nil {
			t.Fatal(err)
		}
	}
	if len(delivered) != 1 || delivered[0].Thinking != "" || delivered[0].Text == "" {
		t.Fatalf("private data escaped to shared output: %+v", delivered)
	}
}
