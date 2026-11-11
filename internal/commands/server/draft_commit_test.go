// Purpose: Verify reconnect snapshots distinguish committed generations from live output.
// Pattern: Observable stream lifecycle regression tests.
// Usage: go test ./internal/commands/server -run TestDraftCommit.
// Related: internal/commands/server/activity.go, internal/commands/server/stream_projection.go.
package server

import (
	"testing"

	"github.com/keonho-kim/meca/internal/agent/record"
)

func TestDraftCommitTracksEachSourceAndResetsForNewGenerations(t *testing.T) {
	for _, kind := range []string{"progress", "team_progress", "tool_started"} {
		t.Run(kind, func(t *testing.T) {
			s, r := projectionFixture()
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
