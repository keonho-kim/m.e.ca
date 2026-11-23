// Purpose: Verify reconnect snapshots distinguish committed generations from live output.
// Pattern: Observable stream lifecycle regression tests.
// Usage: go test ./internal/commands/server -run TestDraftCommit.
// Related: internal/commands/server/activity.go, internal/commands/server/stream_projection.go.
package server

import (
	"fmt"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/team/channel"
	"github.com/keonho-kim/meca/internal/storage/sqlite"
)

func TestDraftsIncludeEveryActiveTeamSource(t *testing.T) {
	s, r := projectionFixture()
	r.teamEnabled = true
	r.originals = map[string]*draftOriginal{}
	t.Cleanup(func() { cleanupDraftFixture(t, r) })
	for i := 1; i <= 17; i++ {
		agent := fmt.Sprintf("Teammate-%02d", i)
		s.emit(r, record.Event{Kind: "team_notice", Agent: agent, Text: "joined"})
		s.emit(r, record.Event{Kind: "stream", Agent: agent, Text: "Working"})
	}
	if drafts := s.State().Drafts; len(drafts) != 17 {
		t.Fatalf("visible drafts = %d, want 17", len(drafts))
	}
	const last = "Teammate-17"
	page, err := s.ReadDraft(t.Context(), r.task.ID, last, r.drafts[last].Generation, 0, 64, false)
	if err != nil || page.Text != "Working" {
		t.Fatalf("last teammate's draft was lost: %+v %v", page, err)
	}
	s.emit(r, record.Event{Kind: "team_notice", Agent: last, Text: "left"})
	if r.originals[last] != nil || r.drafts[last] != nil {
		t.Fatal("departed teammate retained a draft")
	}
}

func TestDraftCommitTracksEachSourceAndResetsForNewGenerations(t *testing.T) {
	for _, kind := range []string{"progress", "team_progress", "tool_started"} {
		t.Run(kind, func(t *testing.T) {
			s, r := projectionFixture()
			t.Cleanup(func() { cleanupDraftFixture(t, r) })
			for _, agent := range []string{"a", "b"} {
				s.emit(r, record.Event{Kind: "stream", Agent: agent, Text: "answer"})
			}
			s.emit(r, record.Event{Kind: kind, Agent: "a", Text: "answer"})
			for _, draft := range s.State().Drafts {
				if draft.Committed != (draft.Agent == "a") {
					t.Fatalf("incorrect committed snapshot: %+v", draft)
				}
			}
			s.emit(r, record.Event{Kind: "generation_started", Agent: "a"})
			for _, draft := range s.State().Drafts {
				if draft.Agent == "a" && draft.Committed {
					t.Fatal("new generation inherited the previous commit")
				}
			}
		})
	}
}

func TestStoredDraftReleasesMemoryAndPagesMatchingHistory(t *testing.T) {
	body := strings.Repeat("text ", 16000)
	dir := t.TempDir()
	catalog, err := sqlite.OpenSessions(dir)
	if err != nil {
		t.Fatal(err)
	}
	store, err := catalog.Create(t.Context(), dir, dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	s := detailFixture(store, nil)
	defer s.cancel()
	s.conversation.Runtime.Session = store.Session()
	r := &request{
		task: Task{ID: "draft-task", Session: s.conversation.Runtime.Session.ID}, conversation: s.conversation,
		drafts: make(map[string]*draftState), originals: make(map[string]*draftOriginal),
	}
	s.active = map[string]*request{r.task.ID: r}
	t.Cleanup(func() { cleanupDraftFixture(t, r) })
	const agent, presentation = "Hana", "request.00000000000000000001"
	s.emit(r, record.Event{Kind: "stream", Agent: agent, Presentation: presentation, Text: body})
	generation := r.drafts[agent].Generation
	saved, err := store.AppendTeamMessage(t.Context(), channel.Message{From: agent, FromStream: true, Text: body, Presentation: presentation})
	if err != nil {
		t.Fatal(err)
	}
	s.emit(r, record.Event{Kind: "team_progress", Agent: agent, Presentation: presentation, Text: body, HistorySequence: saved.Sequence})
	if original := r.originals[agent]; original.text.memory != nil || original.text.historySequence != saved.Sequence || r.drafts[agent].Text != "" {
		t.Fatal("stored generation retained a draft copy or lost its durable reference")
	}
	page, err := s.ReadDraft(t.Context(), r.task.ID, agent, generation, 32000, 640, false)
	if err != nil || page.Text != body[32000:32640] || page.Total != int64(len(body)) {
		t.Fatalf("durable draft page differed from the live original: %d bytes, %v", len(page.Text), err)
	}
	s.emit(r, record.Event{Kind: "generation_started", Agent: agent, Presentation: "request.00000000000000000002"})
	if _, err := s.ReadDraft(t.Context(), r.task.ID, agent, generation, 0, 64, false); err == nil {
		t.Fatal("old durable generation remained readable after reset")
	}
}

func TestActiveDraftSpillsWhenServerMemoryIsReserved(t *testing.T) {
	draftMemory.Lock()
	used := draftMemory.used
	draftMemory.used = draftMemoryBytes
	draftMemory.Unlock()
	defer func() {
		draftMemory.Lock()
		draftMemory.used = used
		draftMemory.Unlock()
	}()
	var text draftText
	defer func() { _ = text.close() }()
	text.append("small original")
	if text.file == nil || text.memory != nil || text.failure != nil {
		t.Fatalf("full memory budget did not spill a small original: %v", text.failure)
	}
	page, err := text.page(6, 8)
	if err != nil || page != "original" {
		t.Fatalf("spilled page: %q %v", page, err)
	}
}
