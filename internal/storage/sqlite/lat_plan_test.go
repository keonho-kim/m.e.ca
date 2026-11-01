// Purpose: Verify one canonical refined plan survives context reload, compaction and export without becoming completed work.
// Pattern: SQLite integration tests.
// Usage: go test ./internal/storage/sqlite -run TestCanonicalPlan.
// Related: internal/agent/lat/plan.go, internal/storage/sqlite/context.go, internal/storage/history/compaction.go.
package sqlite

import (
	"bytes"
	"context"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/compact"
	"github.com/keonho-kim/meca/internal/agent/lat"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

func TestCanonicalPlanRestoresOnceAndRemainsProposedWork(t *testing.T) {
	ctx := context.Background()
	catalog, err := OpenSessions(filepath.Join(t.TempDir(), "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	project := t.TempDir()
	store, err := catalog.Create(ctx, project, project)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if store != nil {
			if err := store.Close(); err != nil {
				t.Error(err)
			}
		}
	}()
	session := store.Session().ID
	request := modelapi.Message{ID: "original_request", Role: "user", Content: "ORIGINAL_REQUEST"}
	if err = store.AppendMessage(ctx, session, request); err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"RAW_INITIAL_PLAN", "RAW_CRITIQUE"} {
		if err = store.RecordEvent(ctx, session, runrecord.Event{Kind: "lat_plan", Text: text, Internal: true}); err != nil {
			t.Fatal(err)
		}
	}
	text := "REFINED_PLAN " + strings.Repeat("inspect source and verify changes\n", 2000)
	if err = store.RecordEvent(ctx, session, runrecord.Event{Kind: "lat_plan", RunID: "parent_run", Text: text}); err != nil {
		t.Fatal(err)
	}
	want := []modelapi.Message{request, lat.Plan(text)}
	for range 2 {
		messages, err := store.ContextMessages(ctx, session, 0, 1<<20, false)
		if err != nil || !reflect.DeepEqual(messages, want) {
			t.Fatalf("context reload duplicated or lost request/plan: count=%d err=%v", len(messages), err)
		}
	}
	if messages, err := store.ContextMessages(ctx, session, 0, 1024, false); err == nil || messages != nil {
		t.Fatal("plan context overflow silently truncated")
	}
	var duplicated int
	if err = store.db.QueryRow("SELECT COUNT(*) FROM tool_events WHERE memory_text<>''").Scan(&duplicated); err != nil || duplicated != 0 {
		t.Fatalf("plan text duplicated into memory_text: count=%d err=%v", duplicated, err)
	}
	mark, err := store.MemoryWatermark(ctx, session, 0)
	if err != nil || mark.Tokens != compact.EstimateBytes(int64(len(request.Content)+len(text))).Amount {
		t.Fatalf("refined plan omitted from occupancy: %+v %v", mark, err)
	}
	records, _, err := store.MemoryRecords(ctx, session, 0, mark.Timeline)
	if err != nil || len(records) != 2 || records[1].Role != "implementation_plan" || records[1].Text != text {
		t.Fatalf("plan treated as completed evidence: count=%d err=%v", len(records), err)
	}
	latMark, err := store.LATWatermark(ctx, session, 0)
	if err != nil || latMark.Bytes != int64(len(request.Content)) {
		t.Fatalf("plan became independent outcome evidence: %+v %v", latMark, err)
	}
	if err = store.Close(); err != nil {
		t.Fatal(err)
	}
	store = nil
	store, err = catalog.Open(ctx, project, session)
	if err != nil {
		t.Fatal(err)
	}
	messages, err := store.ContextMessages(ctx, session, 0, 1<<20, false)
	if err != nil || !reflect.DeepEqual(messages, want) {
		t.Fatalf("restart changed plan context: count=%d err=%v", len(messages), err)
	}
	var audit bytes.Buffer
	if err = store.Export(ctx, session, &audit); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(audit.String(), "RAW_INITIAL_PLAN") || !strings.Contains(audit.String(), "RAW_CRITIQUE") || !strings.Contains(audit.String(), "REFINED_PLAN") {
		t.Fatal("canonical plan or raw internal audit lost from export")
	}
}
