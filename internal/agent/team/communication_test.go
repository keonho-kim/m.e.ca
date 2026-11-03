// Purpose: Verify only committed public speech enables work in a peer's current active lifetime.
// Pattern: Room-state transition regression tests.
// Usage: go test -race ./internal/agent/team -run TestContribution.
// Related: internal/agent/team/peer.go, internal/agent/team/publication.go, internal/tools/team/admission.go.
package team

import (
	"context"
	"errors"
	"testing"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/team/channel"
)

func TestContributionRequiresOwnPublicSpeechInCurrentLifetime(t *testing.T) {
	r := testRoom(t)
	a, b := &Peer{ID: "Hana", room: r}, &Peer{ID: "Ren", room: r}
	for _, event := range []runrecord.Event{
		{Kind: "thinking", Thinking: "I will inspect api/handler.go"},
		{Kind: "stream", Text: "I will inspect api/handler.go"},
		{Kind: "progress", Text: " \n\t"},
	} {
		if err := a.Emit(event); err != nil {
			t.Fatal(err)
		}
		if a.HasSpoken() || b.HasSpoken() {
			t.Fatal("private reasoning, a draft or empty text enabled local work")
		}
	}
	if _, err := r.publish(channel.Message{From: "Ren", Text: "Hana: please inspect the interface"}); err != nil {
		t.Fatal(err)
	}
	if a.HasSpoken() || b.HasSpoken() {
		t.Fatal("an invitation was mistaken for a public contribution")
	}
	if err := a.Emit(runrecord.Event{Kind: "progress", Text: "Ren is editing storage/query.go; I will inspect api/handler.go."}); err != nil {
		t.Fatal(err)
	}
	if !a.HasSpoken() || b.HasSpoken() {
		t.Fatal("public contribution was assigned to the wrong peer")
	}
	r.mu.Lock()
	member := r.active[a.ID].member
	delete(r.active, a.ID)
	r.mu.Unlock()
	r.reserve(member)
	if a.HasSpoken() {
		t.Fatal("historical speech enabled a resumed contribution before its current announcement")
	}
}

type rejectedSpeech struct {
	Repository
	err error
}

func (s rejectedSpeech) AppendTeamMessage(context.Context, channel.Message) (channel.Message, error) {
	return channel.Message{}, s.err
}

func TestContributionPublicationFailureDoesNotEnableWork(t *testing.T) {
	r := testRoom(t)
	want := errors.New("public speech could not be stored")
	r.options.Store = rejectedSpeech{Repository: r.options.Store, err: want}
	a := &Peer{ID: "Hana", room: r}
	if err := a.Emit(runrecord.Event{Kind: "progress", Text: "I will edit api/handler.go."}); !errors.Is(err, want) {
		t.Fatal(err)
	}
	if a.HasSpoken() {
		t.Fatal("failed publication enabled local work")
	}
}
