// Purpose: Preserve one terminal result per failed or skipped native call.
// Pattern: Batch event contract regression.
// Usage: go test ./internal/agent/run -run TestBatchFailure.
// Related: internal/agent/run/batch_wave.go, internal/agent/run/batch.go.
package run

import (
	"context"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"testing"
)

func TestBatchFailureHasOneEventPerCall(t *testing.T) {
	calls := []toolapi.Call{
		batchCall("missing", "edit_file", map[string]any{"path": "missing", "edits": []map[string]any{{"old_text": "before", "new_text": "after"}}}),
		batchCall("other", "write_file", map[string]any{"path": "other", "content": "new"}),
	}
	r, _, _, j, _ := realBatchRunner(t, calls)
	if _, err := r.Run(context.Background(), "inspect files", approve, emit); err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, e := range j.events {
		if e.Kind == "tool_failed" {
			counts[e.CallID]++
		}
	}
	for _, call := range calls {
		if counts[call.ID] != 1 {
			t.Fatalf("terminal events = %+v", counts)
		}
	}
}
