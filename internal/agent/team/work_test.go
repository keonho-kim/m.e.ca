// Purpose: Share actual work, reject unseen overlapping edits and retain compact request-owned state.
// Pattern: Room integration tests without model or filesystem I/O.
// Usage: Team tests exercise declaration, observations and decision boundaries.
// Related: internal/agent/team/work.go, internal/agent/team/observations.go.
package team

import (
	"strings"
	"testing"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func TestWorkRequiresCoordinationAndReviewOfRelatedActualChanges(t *testing.T) {
	r := testRoom(t)
	a, b := &Peer{ID: "Hana", room: r}, &Peer{ID: "Ren", room: r}
	if _, err := a.SetWork("Implement API", []string{"/project/api"}, "", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := b.SetWork("Duplicate API", []string{"/project/api/file"}, "", 0); err == nil || !strings.Contains(err.Error(), "Hana") {
		t.Fatal("conflict did not identify owner", err)
	}
	if err := a.Emit(runrecord.Event{Kind: "tool_succeeded", Operation: &toolapi.Operation{Name: "write_file", Status: "tool_succeeded", Started: true, Changes: []toolapi.FileChange{{Path: "/project/api/file", Action: "modified"}}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := a.SetWork("Ready for Ren to integrate", nil, "", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := b.SetWork("Integrate Hana's API", []string{"/project/api", "/project/tests"}, "", 0); err != nil {
		t.Fatal(err)
	}
	if err := b.CheckWork([]string{"/project/api/file"}); err == nil {
		t.Fatal("unseen actual edit did not invalidate the old candidate")
	}
	if err := b.CheckWork([]string{"/project/tests/file"}); err != nil {
		t.Fatal("unrelated update blocked independent work", err)
	}
	batch, err := b.Inbox(r.ctx)
	if err != nil {
		t.Fatal(err)
	}
	saw := false
	for _, message := range batch.Messages {
		saw = saw || strings.Contains(message.Content, "write_file tool_succeeded\nmodified /project/api/file")
	}
	if !saw {
		t.Fatal("actual work did not reach teammate context")
	}
	b.BeginDecision()
	if err := b.CheckWork([]string{"/project/api/file"}); err != nil {
		t.Fatal("reviewed handoff blocked", err)
	}
	if !strings.Contains(b.ContextObservations(nil), "Ren: Integrate Hana's API") {
		t.Fatal("current scope absent from post-compaction observations")
	}
}

func TestCommandFactsCoalesceStatesAndNeverCopyPrivateOutput(t *testing.T) {
	r := testRoom(t)
	p := &Peer{ID: "Hana", room: r}
	op := &toolapi.Operation{Name: "execute_command", Status: "tool_succeeded", Command: "run checks", Process: &toolapi.ProcessOutcome{CommandID: "cmd_own", Status: "running"}}
	event := runrecord.Event{Kind: "tool_succeeded", Operation: op, Thinking: "private thoughts", Output: []byte(`{"Stdout":"private verbose log"}`)}
	for range 3 {
		if err := p.Emit(event); err != nil {
			t.Fatal(err)
		}
	}
	op.Process = &toolapi.ProcessOutcome{CommandID: "cmd_own", Status: "exited", ExitCode: 0}
	if err := p.Emit(event); err != nil {
		t.Fatal(err)
	}
	messages, err := (&Peer{ID: "Ren", room: r}).ReadMessages(r.ctx, 0)
	if err != nil || len(messages) != 2 {
		t.Fatal("unchanged process status duplicated", len(messages), err)
	}
	for _, value := range messages {
		if !value.Observation || strings.Contains(value.Text, "private") {
			t.Fatal("private payload entered public facts")
		}
	}
}

func TestUnchangedWorkDoesNotWakePeersOrDuplicatePublicRecords(t *testing.T) {
	r := testRoom(t)
	p := &Peer{ID: "Hana", room: r}
	for range 3 {
		if _, err := p.SetWork("Own API", []string{"/project/api"}, "", 0); err != nil {
			t.Fatal(err)
		}
	}
	for range 3 {
		if _, err := p.SetWork("API handoff ready", nil, "", 0); err != nil {
			t.Fatal(err)
		}
	}
	messages, err := p.ReadMessages(r.ctx, 0)
	if err != nil || len(messages) != 2 {
		t.Fatal("unchanged work produced repeated conversation", len(messages), err)
	}
}

func TestWorkDeclarationIsAdmittedOnlyToOtherMembers(t *testing.T) {
	r := testRoom(t)
	p := &Peer{ID: "Hana", room: r}
	if _, err := p.SetWork("Own API", []string{"/project/api"}, "", 0); err != nil {
		t.Fatal(err)
	}
	own, err := p.Inbox(r.ctx)
	if err != nil || len(own.Messages) != 0 || own.Cursor == 0 {
		t.Fatal("native declaration returned to its author", own, err)
	}
	other, err := (&Peer{ID: "Ren", room: r}).Inbox(r.ctx)
	if err != nil || len(other.Messages) != 1 || !strings.Contains(other.Messages[0].Content, "Own API") {
		t.Fatal("declaration did not reach the other member", other, err)
	}
}
