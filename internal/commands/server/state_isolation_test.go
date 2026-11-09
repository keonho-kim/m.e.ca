// Purpose: Probe session attribution and failed draft storage accounting.
// Pattern: Application ownership regression tests.
// Usage: Run go test for this package; failures retain task and storage ownership.
// Related: internal/commands/server/state_page.go, internal/commands/server/draft_spool.go.
package server

import (
	"fmt"
	"github.com/keonho-kim/meca/internal/agent/record"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"path/filepath"
	"strings"
	"testing"
)

func TestMismatchedSessionTaskState(t *testing.T) {
	s := detailFixture(&detailHistory{}, nil)
	defer s.Close()
	first := s.conversation
	second := &Conversation{History: &detailHistory{}, Close: func() error { return nil }}
	second.Runtime.Session.ID = "session_BBBBBBBBBBBBBBBBBBBBBBBBBB"
	s.jobs = map[string]*Conversation{second.Runtime.Session.ID: second}
	s.active = map[string]*request{"task_b": {task: Task{ID: "task_b", Session: second.Runtime.Session.ID, Status: "running"}, conversation: second}}
	state, err := s.StatePage(t.Context(), first.Runtime.Session.ID, "task_b", "", 32)
	if faults.Code(err) != diagcatalog.InvalidArgument {
		t.Fatalf("mismatched session and task accepted: %+v, %v", state, err)
	}
}

func BenchmarkFilteredState(b *testing.B) {
	for _, count := range []int{1, 128, 4096} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			s, _ := projectionFixture()
			s.active = make(map[string]*request, count)
			for i := range count {
				id := fmt.Sprintf("task-%05d", i)
				s.active[id] = &request{task: Task{ID: id}}
			}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				_ = s.snapshotPage("task-00000", "", 1)
			}
		})
	}
}

func TestRejoinedAgentRejectsOldDraftGeneration(t *testing.T) {
	s, r := projectionFixture()
	r.originals = map[string]*draftOriginal{}
	defer r.closeDrafts()
	s.emit(r, record.Event{Kind: "generation_started", Agent: "Hana", RunID: "old"})
	s.emit(r, record.Event{Kind: "stream", Agent: "Hana", RunID: "old", Text: strings.Repeat("old", 100)})
	previous := r.drafts["Hana"].Generation
	s.emit(r, record.Event{Kind: "team_notice", Agent: "Hana", Text: "left"})
	s.emit(r, record.Event{Kind: "generation_started", Agent: "Hana", RunID: "new"})
	s.emit(r, record.Event{Kind: "stream", Agent: "Hana", RunID: "new", Text: strings.Repeat("new", 100)})
	page, err := s.ReadDraft(t.Context(), r.task.ID, "Hana", previous, 3, 16, false)
	if faults.Code(err) != diagcatalog.Conflict {
		t.Fatalf("old generation %d accepted: %q %v", previous, page.Text, err)
	}
}

func TestTargetedSettingEventSession(t *testing.T) {
	s := detailFixture(&detailHistory{}, nil)
	defer s.Close()
	first := s.conversation
	second := &Conversation{History: &detailHistory{}, Close: func() error { return nil }}
	second.Runtime.Session.ID = "session_BBBBBBBBBBBBBBBBBBBBBBBBBB"
	s.conversation = second
	s.jobs = map[string]*Conversation{first.Runtime.Session.ID: first}
	cursor := s.State().Cursor
	enabled := false
	if _, err := s.Configure(t.Context(), first.Runtime.Session.ID, nil, &enabled); err != nil {
		t.Fatal(err)
	}
	batch, err := s.Replay(cursor)
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range batch.Events {
		if event.Record.Kind == "settings_changed" && event.Session != first.Runtime.Session.ID {
			t.Fatalf("setting for %s broadcast as %s", first.Runtime.Session.ID, event.Session)
		}
	}
}

func TestFailedDraftSpoolReturnsCapacity(t *testing.T) {
	t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "absent"))
	draftDisk.Lock()
	before := draftDisk.used
	draftDisk.Unlock()
	var draft draftText
	defer draft.close()
	draft.append(strings.Repeat("x", maxDraftBytes+1))
	if draft.failure == nil {
		t.Fatal("expected temporary storage failure")
	}
	draftDisk.Lock()
	after := draftDisk.used
	draftDisk.Unlock()
	if after != before {
		t.Fatalf("failed spool without a file retained %d reserved bytes", after-before)
	}
}
