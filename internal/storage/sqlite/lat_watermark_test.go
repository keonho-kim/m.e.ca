// Purpose: Separate scanned history from factual evidence so internal LAT events are not rescanned.
// Pattern: Watermark boundary and append regression.
// Usage: Run TestLATWatermarkSkipsInternalSuffix.
// Related: internal/storage/sqlite/lat_sources.go, internal/agent/lat/memory/controller.go.
package sqlite

import (
	"context"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

func TestLATWatermarkSkipsInternalSuffix(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx, id := context.Background(), s.Session().ID
	if err := s.AppendMessage(ctx, id, modelapi.Message{Role: "user", Content: "First request"}); err != nil {
		t.Fatal(err)
	}
	first, err := s.LATWatermark(ctx, id, 0)
	if err != nil {
		t.Fatal(err)
	}
	for range 20 {
		if err := s.RecordEvent(ctx, id, runrecord.Event{Kind: "thinking", Thinking: "private", Internal: true, LAT: &runrecord.LATRun{ID: "lat_plan", ParentRunID: "run_main", Phase: "plan"}}); err != nil {
			t.Fatal(err)
		}
	}
	skipped, err := s.LATWatermark(ctx, id, first.Scanned)
	if err != nil || skipped.Bytes != 0 || skipped.Through != 0 || skipped.Scanned <= first.Scanned {
		t.Fatalf("invalid skipped suffix: %+v %v", skipped, err)
	}
	if err := s.AppendMessage(ctx, id, modelapi.Message{Role: "user", Content: "New request"}); err != nil {
		t.Fatal(err)
	}
	latest, err := s.LATWatermark(ctx, id, skipped.Scanned)
	if err != nil || latest.Bytes != int64(len("New request")) || latest.Through <= skipped.Scanned {
		t.Fatalf("new instruction skipped: %+v %v", latest, err)
	}
}
