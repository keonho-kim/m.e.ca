// Purpose: Share large objectives with message payloads while preserving rollback, reopening and exact recovery.
// Pattern: Canonical persistence integration tests.
// Usage: go test ./internal/storage/sqlite -run TestCheckpointObjective.
// Related: internal/storage/sqlite/state_objective.go, internal/storage/sqlite/run_state.go.
package sqlite

import (
	"context"
	"strings"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

func TestCheckpointObjectiveReusesOriginalPayload(t *testing.T) {
	for _, objective := range []string{strings.Repeat("review source\n", 10000), `{"source":"` + strings.Repeat("x", 100000) + `"}`} {
		s, err := newTestStore(t)
		if err != nil {
			t.Fatal(err)
		}
		id := s.Session().ID
		if err := s.AppendMessage(t.Context(), id, modelapi.Message{Role: "user", Content: objective}); err != nil {
			t.Fatal(err)
		}
		cp := runrecord.Checkpoint{ID: "run_objective", Objective: objective, Status: runrecord.Running, Phase: "decision"}
		for i := range 12 {
			cp.ModelCalls = i
			if err := s.RecordEvent(t.Context(), id, runrecord.Event{Kind: "run_checkpoint", Checkpoint: &cp}); err != nil {
				t.Fatal(err)
			}
		}
		var count, raw int
		if err := s.db.QueryRow("SELECT COUNT(*) FROM payloads").Scan(&count); err != nil {
			t.Fatal(err)
		}
		if err := s.db.QueryRow("SELECT raw_bytes FROM run_states").Scan(&raw); err != nil {
			t.Fatal(err)
		}
		if count != 1 || raw > 4096 {
			t.Fatal("checkpoint duplicated original objective", count, raw)
		}
		// Use another handle so recovery cannot rely on the preparation cache.
		db, err := connect(t.Context(), s.path, true)
		if err != nil {
			t.Fatal(err)
		}
		reader := &Store{db: db, path: s.path, session: s.session}
		got, err := reader.LatestRun(t.Context(), id)
		if err != nil || got == nil || got.Objective != objective || got.ModelCalls != 11 {
			t.Fatal("objective recovery failed", err)
		}
		if err := reader.Close(); err != nil {
			t.Fatal(err)
		}
		if err := s.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCheckpointObjectiveCacheDoesNotAssumeCommit(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	cp := runrecord.Checkpoint{ID: "run_retry", Objective: strings.Repeat("uncommitted objective\n", 10000), Status: runrecord.Running, Phase: "decision"}
	event := runrecord.Event{Kind: "run_checkpoint", Checkpoint: &cp}
	encoder := journalEncoder{store: s, ctx: context.Background(), lookup: s.db}
	if _, err := encoder.prepareRunState(event); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordEvent(t.Context(), s.Session().ID, event); err != nil {
		t.Fatal("unpublished cache prevented retry", err)
	}
	got, err := s.LatestRun(t.Context(), s.Session().ID)
	if err != nil || got == nil || got.Objective != cp.Objective {
		t.Fatal("retry lost objective", err)
	}
}
