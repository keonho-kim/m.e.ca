// Purpose: Preserve large grounding reasons through event and checkpoint persistence.
// Pattern: Repository boundary regression test.
// Usage: go test ./internal/storage/sqlite -run TestLargeGroundingReason.
// Related: internal/storage/sqlite/events.go, internal/storage/sqlite/runs.go.
package sqlite

import (
	"context"
	"github.com/keonho-kim/v-code/internal/agent/grounding"
	"github.com/keonho-kim/v-code/internal/agent/run"
	"github.com/keonho-kim/v-code/internal/agent/tool"
	"strings"
	"testing"
)

func TestLargeGroundingReason(t *testing.T) {
	store, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Error(err)
		}
	})
	reason := strings.Repeat("<", 17*1024*1024)
	record, err := grounding.Bind(tool.Call{ID: "large", Name: "inspect_fixture", Arguments: []byte(`{"path":"file.txt"}`)}, reason, "run")
	if err != nil {
		t.Fatal(err)
	}
	cp := run.Checkpoint{ID: "run", Objective: "Read", Status: run.Running, Phase: "grounding", InstructionsDigest: strings.Repeat("a", 64), Grounding: &record}
	ctx := context.Background()
	if err := store.RecordEvent(ctx, store.Session().ID, run.Event{Kind: "run_checkpoint", Checkpoint: &cp}); err != nil {
		t.Fatal(err)
	}
	recovered, err := store.LatestRun(ctx, store.Session().ID)
	if err != nil {
		t.Fatal(err)
	}
	if recovered.Grounding == nil || recovered.Grounding.Reason != reason {
		t.Fatal("grounding reason was lost or changed")
	}
}
