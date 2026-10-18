// Purpose: Preserve interrupted checkpoints and outcome claims without executing stored drafts.
// Pattern: Repository recovery tests.
// Usage: go test ./internal/storage/sqlite -run TestRunCheckpoint.
// Related: internal/storage/sqlite/runs.go, internal/storage/recovery/recovery.go.
package sqlite

import (
	"context"
	"encoding/json"
	"github.com/keonho-kim/v-code/internal/agent/completion"
	"github.com/keonho-kim/v-code/internal/agent/run"
	"github.com/keonho-kim/v-code/internal/storage/recovery"
	"strings"
	"testing"
)

func TestRunCheckpointRecovery(t *testing.T) {
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
	cp := run.Checkpoint{Work: completion.State{Receipts: []completion.Receipt{{ID: "call_1", Tool: "inspect_fixture", Success: true, Output: "Read existing contents"}}}, ReportReason: "reporting requested", LastChange: "Created file /project/empty (0 bytes)", ID: "task", Objective: "edit; preserve existing file contents", Status: run.Running, Phase: "grounding", Tool: "write_file", InstructionsDigest: strings.Repeat("a", 64), Arguments: map[string]json.RawMessage{"path": json.RawMessage(`"file.txt"`)}}
	if err = s.RecordEvent(ctx, session.ID, run.Event{Kind: "run_checkpoint", Checkpoint: &cp}); err != nil {
		t.Fatal(err)
	}
	if err = recovery.Restore(ctx, s, session.ID); err != nil {
		t.Fatal(err)
	}
	recovered, err := s.LatestRun(ctx, session.ID)
	if err != nil || recovered == nil || recovered.Status != run.Interrupted || recovered.Tool != "write_file" || recovered.Objective != cp.Objective || recovered.LastChange != cp.LastChange || len(recovered.Work.Receipts) != 1 || recovered.Work.Receipts[0].ID != "call_1" || recovered.ReportReason != cp.ReportReason {
		t.Fatalf("recovery: %+v %v", recovered, err)
	}
	history, err := s.Messages(ctx, session.ID, 100)
	if err != nil || len(history) != 0 {
		t.Fatalf("draft was replayed: %+v %v", history, err)
	}
}
