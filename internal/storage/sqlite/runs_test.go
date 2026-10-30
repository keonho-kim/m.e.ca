// Purpose: Preserve interrupted checkpoints and outcome claims without executing stored drafts.
// Pattern: Repository recovery tests.
// Usage: go test ./internal/storage/sqlite -run TestRunCheckpoint.
// Related: internal/storage/sqlite/runs.go, internal/storage/recovery/recovery.go.
package sqlite

import (
	"context"
	"encoding/json"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/storage/recovery"
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
	cp := runrecord.Checkpoint{Work: tool.Facts{Files: []tool.FileFact{{Path: "file.txt", Version: "v1"}}}, ReportReason: "reporting requested", LastChange: "Created file /project/empty (0 bytes)", ID: "task", Objective: "edit; preserve existing file contents", Status: runrecord.Running, Phase: "decision", Tool: "write_file", InstructionsDigest: strings.Repeat("a", 64), Arguments: map[string]json.RawMessage{"path": json.RawMessage(`"file.txt"`)}}
	if err = s.RecordEvent(ctx, session.ID, runrecord.Event{Kind: "run_checkpoint", Checkpoint: &cp}); err != nil {
		t.Fatal(err)
	}
	if err = recovery.Restore(ctx, s, session.ID); err != nil {
		t.Fatal(err)
	}
	recovered, err := s.LatestRun(ctx, session.ID)
	if err != nil || recovered == nil || recovered.Status != runrecord.Interrupted || recovered.Tool != "write_file" || recovered.Objective != cp.Objective || recovered.LastChange != cp.LastChange || len(recovered.Work.Files) != 1 || recovered.Work.Files[0].Path != "file.txt" || recovered.ReportReason != cp.ReportReason {
		t.Fatalf("recovery: %+v %v", recovered, err)
	}
	history, err := s.ContextMessages(ctx, session.ID, 0, 16*1024*1024, true)
	if err != nil || len(history) != 0 {
		t.Fatalf("draft was replayed: %+v %v", history, err)
	}
}
