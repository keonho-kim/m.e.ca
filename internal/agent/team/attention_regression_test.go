// Purpose: Keep wording and repeated observations visible without renewing peer inference.
// Pattern: Deterministic publication and lifecycle regressions.
// Usage: Team tests exercise shared notification decisions without model calls.
// Related: internal/agent/team/attention.go, internal/agent/team/observations.go.
package team

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	record "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/tool"
)

func TestWorkWordingDoesNotDeferAnotherPeersDeparture(t *testing.T) {
	r := testRoom(t)
	a, b := &Peer{ID: "Hana", room: r}, &Peer{ID: "Ren", room: r}
	if _, err := a.SetWork("API", []string{"/project/api"}, "UserID is a string", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := b.SetWork("Docs", []string{"/project/docs"}, "", 0); err != nil {
		t.Fatal(err)
	}
	b.Inbox(r.ctx)
	b.BeginDecision()
	for i := range 3 {
		if _, err := a.SetWork(fmt.Sprintf("API wording %d", i), []string{"/project/api"}, "UserID is a string", 0); err != nil {
			t.Fatal(err)
		}
	}
	batch, err := b.Inbox(r.ctx)
	if err != nil || batch.Reconsider || len(batch.Messages) != 3 {
		t.Fatal("wording either disappeared or forced another decision", batch, err)
	}
	if left, err := b.Leave(r.ctx, "Documentation complete"); !left || err != nil {
		t.Fatal("wording delayed a completed contribution", err)
	}
}

func TestContractChangesIncludingRemovalRequireReview(t *testing.T) {
	r := testRoom(t)
	a, b := &Peer{ID: "Hana", room: r}, &Peer{ID: "Ren", room: r}
	a.SetWork("API", []string{"/project/api"}, "UserID is a string", 0)
	b.SetWork("Docs", []string{"/project/docs"}, "", 0)
	for _, contract := range []string{"UserID is an integer", ""} {
		b.Inbox(r.ctx)
		b.BeginDecision()
		if _, err := a.SetWork("API", []string{"/project/api"}, contract, 0); err != nil {
			t.Fatal(err)
		}
		batch, err := b.Inbox(r.ctx)
		if err != nil || !batch.Reconsider {
			t.Fatal("changed or removed interface failed to reach a teammate", err)
		}
	}
}

func TestRepeatedFailureRemainsVisibleWithoutRestartingIdlePeers(t *testing.T) {
	r := testRoom(t)
	a, b := &Peer{ID: "Hana", room: r}, &Peer{ID: "Ren", room: r}
	emit := func(id, diagnostic string) {
		t.Helper()
		output, _ := json.Marshal(map[string]string{"Stdout": "", "Stderr": diagnostic})
		err := a.Emit(record.Event{Kind: "tool_failed", Output: output, Operation: &tool.Operation{
			Name: "execute_command", Status: "tool_failed", Started: true, Command: "check",
			Process: &tool.ProcessOutcome{CommandID: id, Status: "exited", ExitCode: 1},
			Failure: &tool.Failure{Code: tool.ProcessFailed, Detail: diagnostic},
		}})
		if err != nil {
			t.Fatal(err)
		}
	}
	emit("first", "same failure")
	if batch, err := b.Inbox(r.ctx); err != nil || !batch.Reconsider {
		t.Fatal("new failure did not request a decision", err)
	}
	b.BeginDecision()
	emit("second", "same failure")
	batch, err := b.Inbox(r.ctx)
	if err != nil || batch.Reconsider || len(batch.Messages) != 1 {
		t.Fatal("repeated failure was hidden or renewed attention", batch, err)
	}
	r.mu.Lock()
	for _, live := range r.active {
		live.waiting = true
	}
	r.idleAttempted, r.idleEpoch = true, r.progress
	_, idleErr := r.resolveIdleLocked()
	r.mu.Unlock()
	if idleErr == nil || !strings.Contains(idleErr.Error(), "team stalled") {
		t.Fatal("repeated failure bypassed collective idle detection", idleErr)
	}
	emit("third", "a different failure")
	if next, err := b.Inbox(r.ctx); err != nil || !next.Reconsider {
		t.Fatal("new failure was mistaken for an old observation", err)
	}
}

func TestRepeatedFileEffectsStillRequireFreshReview(t *testing.T) {
	r := testRoom(t)
	a, b := &Peer{ID: "Hana", room: r}, &Peer{ID: "Ren", room: r}
	for range 2 {
		b.Inbox(r.ctx)
		b.BeginDecision()
		event := record.Event{Kind: "tool_succeeded", Operation: &tool.Operation{
			Name: "write_file", Status: "tool_succeeded",
			Changes: []tool.FileChange{{Path: "/project/api.go", Action: "modified", BeforeDigest: "old", AfterDigest: "new"}},
		}}
		if err := a.Emit(event); err != nil {
			t.Fatal(err)
		}
		if batch, err := b.Inbox(r.ctx); err != nil || !batch.Reconsider {
			t.Fatal("repeated content hid a newly observed file effect", err)
		}
	}
}

func TestWorkAnswerStillNotifiesItsQuestionerWithoutChangingContract(t *testing.T) {
	r := testRoom(t)
	a, b := &Peer{ID: "Hana", room: r}, &Peer{ID: "Ren", room: r}
	b.SetWork("API", []string{"/project/api"}, "UserID is a string", 0)
	question, err := a.Discuss(r.ctx, "Is UserID a string?", 0, []string{b.ID})
	if err != nil {
		t.Fatal(err)
	}
	for _, peer := range []*Peer{a, b} {
		peer.Inbox(r.ctx)
		peer.BeginDecision()
	}
	if _, err := b.SetWork("Confirmed: UserID remains a string", []string{"/project/api"}, "UserID is a string", question); err != nil {
		t.Fatal(err)
	}
	if batch, err := a.Inbox(r.ctx); err != nil || !batch.Reconsider {
		t.Fatal("unchanged contract suppressed a direct answer", err)
	}
}
