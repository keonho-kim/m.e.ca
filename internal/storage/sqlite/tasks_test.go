// Purpose: Verify interrupted argument collection is recovered as a checkpoint without executing a draft.
// Pattern: Repository recovery tests.
// Usage: go test ./internal/storage/sqlite -run TestTaskCheckpoint.
// Related: internal/storage/sqlite/tasks.go, internal/storage/recovery/recovery.go.
package sqlite

import (
	"context"
	"encoding/json"
	"github.com/keonho-kim/v-code/internal/agent/run"
	"github.com/keonho-kim/v-code/internal/storage/recovery"
	"strings"
	"testing"
)

func TestTaskCheckpointRecovery(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	})
	ctx := context.Background()
	session := s.Session()
	cp := run.Checkpoint{ID: "task", Objective: "edit", Status: run.Running, Phase: "argument", Tool: "write_file", Parameter: "content", InstructionsDigest: strings.Repeat("a", 64), Arguments: map[string]json.RawMessage{"path": json.RawMessage(`"file.txt"`)}}
	if err = s.RecordEvent(ctx, session.ID, run.Event{Kind: "task_checkpoint", Checkpoint: &cp}); err != nil {
		t.Fatal(err)
	}
	if err = recovery.Restore(ctx, s, session.ID); err != nil {
		t.Fatal(err)
	}
	recovered, err := s.LatestTask(ctx, session.ID)
	if err != nil || recovered == nil || recovered.Status != run.Interrupted || recovered.Parameter != "content" {
		t.Fatalf("recovery: %+v %v", recovered, err)
	}
	history, err := s.Messages(ctx, session.ID, 100)
	if err != nil || len(history) != 0 {
		t.Fatalf("draft was replayed: %+v %v", history, err)
	}
}
