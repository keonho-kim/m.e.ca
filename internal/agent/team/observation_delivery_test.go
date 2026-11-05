// Purpose: Preserve shared facts without waking peers for passive status or retaining completed commands.
// Pattern: Deterministic room and command-lifecycle regression tests.
// Usage: Team tests cover notification, admission and terminal metadata ownership together.
// Related: internal/agent/team/observations.go, internal/agent/team/inbox.go.
package team

import (
	"context"
	"fmt"
	"strings"
	"testing"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/team/channel"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func TestPassiveCommandFactsReachPeersWithoutRequestingInference(t *testing.T) {
	r := testRoom(t)
	p, other := &Peer{ID: "Hana", room: r}, &Peer{ID: "Ren", room: r}
	op := &toolapi.Operation{Name: "execute_command", Status: "tool_succeeded", Command: "cat README.md", Process: &toolapi.ProcessOutcome{CommandID: "one", Status: "exited"}}
	if err := p.Emit(runrecord.Event{Kind: "tool_succeeded", Operation: op}); err != nil {
		t.Fatal(err)
	}
	if len(r.active[other.ID].wake) != 0 {
		t.Fatal("a successful inspection woke an idle teammate")
	}
	if r.active[other.ID].attention != 0 {
		t.Fatal("passive status requested another model call")
	}
	batch, err := other.Inbox(r.ctx)
	if err != nil || len(batch.Messages) != 0 || batch.Cursor == 0 || !strings.Contains(other.ContextObservations(nil), "cat README.md") {
		t.Fatal("the next decision lost the passive observation", batch, err)
	}
	op = &toolapi.Operation{Name: "write_file", Status: "tool_succeeded", Changes: []toolapi.FileChange{{Path: "/project/api.go", Action: "modified"}}}
	if err := p.Emit(runrecord.Event{Kind: "tool_succeeded", Operation: op}); err != nil {
		t.Fatal(err)
	}
	batch, err = other.Inbox(r.ctx)
	if err != nil || len(batch.Messages) != 1 || len(r.active[other.ID].wake) != 1 {
		t.Fatal("actual changes stopped notifying teammates", batch, err)
	}
}

type checkedObservationStore struct {
	*memoryRoom
	referenced bool
}

func (s *checkedObservationStore) AppendTeamMessage(ctx context.Context, message channel.Message) (channel.Message, error) {
	if err := ctx.Err(); err != nil {
		return channel.Message{}, err
	}
	s.referenced = message.Reference != nil
	return s.memoryRoom.AppendTeamMessage(ctx, message)
}

func TestCommittedResultKeepsItsGroupedReferenceAfterCancellation(t *testing.T) {
	r := testRoom(t)
	store := &checkedObservationStore{memoryRoom: r.options.Store.(*memoryRoom)}
	r.options.Store = store
	separateWrites := 0
	r.options.Record = func(context.Context, runrecord.Event) error { separateWrites++; return nil }
	r.cancel(context.Canceled)
	p := &Peer{ID: "Hana", room: r}
	event := runrecord.Event{Kind: "tool_succeeded", Reference: &runrecord.EventReference{Session: "session_private", Sequence: 1},
		Operation: &toolapi.Operation{Name: "write_file", Status: "tool_succeeded", Changes: []toolapi.FileChange{{Path: "/project/api.go", Action: "modified"}}}}
	if err := p.Emit(event); err != nil || !store.referenced || separateWrites != 0 {
		t.Fatal("committed result lost its reference or wrote it separately", store.referenced, separateWrites, err)
	}
}

func TestCommandFailureAndCancellationStillWakeTeammates(t *testing.T) {
	for _, terminal := range []struct {
		name, status, outcome string
		exit                  int
	}{
		{"nonzero", "exited", "tool_succeeded", 1},
		{"cancelled", "cancelled", "tool_succeeded", 0},
		{"failed", "exited", "tool_failed", 0},
	} {
		t.Run(terminal.name, func(t *testing.T) {
			r := testRoom(t)
			p := &Peer{ID: "Hana", room: r}
			op := &toolapi.Operation{Name: "execute_command", Status: terminal.outcome,
				Process: &toolapi.ProcessOutcome{CommandID: "one", Status: terminal.status, ExitCode: terminal.exit}}
			if err := p.Emit(runrecord.Event{Kind: terminal.outcome, Operation: op}); err != nil {
				t.Fatal(err)
			}
			if len(r.active["Ren"].wake) != 1 {
				t.Fatal("a failure or cancellation was treated as passive status")
			}
		})
	}
}

func TestCommandObservationsReleaseTerminalIdentityAndIgnoreArchivePolling(t *testing.T) {
	r := testRoom(t)
	p := &Peer{ID: "Hana", room: r}
	for i := range 128 {
		op := &toolapi.Operation{Name: "execute_command", Status: "tool_succeeded", Process: &toolapi.ProcessOutcome{CommandID: fmt.Sprint(i), Status: "exited"}}
		if err := p.Emit(runrecord.Event{Kind: "tool_succeeded", Operation: op}); err != nil {
			t.Fatal(err)
		}
	}
	if r.active[p.ID].statuses != nil {
		t.Fatal("synchronously completed commands retained their identities")
	}
	op := &toolapi.Operation{Name: "execute_command", Status: "tool_succeeded", Process: &toolapi.ProcessOutcome{CommandID: "background", Status: "running"}}
	if err := p.Emit(runrecord.Event{Kind: "tool_succeeded", Operation: op}); err != nil {
		t.Fatal(err)
	}
	op.Name = "wait_command"
	for range 3 {
		if err := p.Emit(runrecord.Event{Kind: "tool_succeeded", Operation: op}); err != nil {
			t.Fatal(err)
		}
	}
	op.Process = &toolapi.ProcessOutcome{CommandID: "background", Status: "exited"}
	if err := p.Emit(runrecord.Event{Kind: "tool_succeeded", Operation: op}); err != nil {
		t.Fatal(err)
	}
	through := r.latestPublic
	for range 3 {
		if err := p.Emit(runrecord.Event{Kind: "tool_succeeded", Operation: op}); err != nil {
			t.Fatal(err)
		}
	}
	if r.active[p.ID].statuses != nil || r.latestPublic != through || through != 130 {
		t.Fatal("terminal state was retained or archive polling was rebroadcast", through, r.latestPublic)
	}
}
