// Purpose: Verify independent memo persistence and original-source selection across main compaction.
// Pattern: SQLite integration and integrity tests.
// Usage: go test ./internal/storage/sqlite -run TestLATMemory.
// Related: internal/storage/sqlite/lat_memory.go, internal/storage/sqlite/lat_sources.go.
package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/compact"
	latmemory "github.com/keonho-kim/meca/internal/agent/lat/memory"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func TestLATMemoryReadsOriginalRequirementsAndEffects(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx, id := context.Background(), s.Session().ID
	for i := range 12 {
		if err = s.AppendMessage(ctx, id, modelapi.Message{Role: "user", Content: fmt.Sprintf("REQUIREMENT_%02d", i)}); err != nil {
			t.Fatal(err)
		}
	}
	if err = s.AppendMessage(ctx, id, modelapi.Message{Role: "assistant", Content: "PRIVATE_MAIN_ANSWER", Reasoning: "PRIVATE_THINKING"}); err != nil {
		t.Fatal(err)
	}
	if err = s.RecordEvent(ctx, id, runrecord.Event{Kind: "lat_feedback", Text: "PRIVATE_PRIOR_VERDICT"}); err != nil {
		t.Fatal(err)
	}
	call := toolapi.Call{ID: "write", Name: "write_file", Arguments: json.RawMessage(`{"path":"a","content":"` + strings.Repeat("HIDDEN_ARGUMENT", 10000) + `"}`)}
	event := runrecord.Event{Kind: "tool_succeeded", CallID: call.ID, Call: &call, Text: "actual change", Output: json.RawMessage(`{"Changed":true}`), ChangePreview: "+ACTUAL_SOURCE_CHANGE", Operation: &toolapi.Operation{Name: "write_file", Status: "tool_succeeded", Result: "ACTUAL_RESULT", Changes: []toolapi.FileChange{{Path: "a", Action: "write"}}}}
	if err = s.RecordEvent(ctx, id, event); err != nil {
		t.Fatal(err)
	}
	mark, err := s.LATWatermark(ctx, id, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.InstallMemory(ctx, id, 0, compact.Memory{Revision: 1, Through: mark.Through, Markdown: "PRIVATE_MAIN_SUMMARY"}); err != nil {
		t.Fatal(err)
	}
	// Corrupt only the large hidden argument. The independent projection must
	// still work because it uses actual outcome fields, not original arguments.
	if _, err = s.db.Exec("UPDATE payloads SET body=X'00'"); err != nil {
		t.Fatal(err)
	}
	records, cursor, err := s.LATRecords(ctx, id, 0, mark.Through, latmemory.MaxBytes)
	if err != nil || len(records) != 13 || cursor != mark.Through {
		t.Fatalf("sources=%d cursor=%d error=%v", len(records), cursor, err)
	}
	var all strings.Builder
	for _, record := range records {
		all.WriteString(record.Text)
	}
	for _, value := range []string{"REQUIREMENT_00", "REQUIREMENT_11", "ACTUAL_RESULT", "ACTUAL_SOURCE_CHANGE"} {
		if !strings.Contains(all.String(), value) {
			t.Fatal("original source lost", value)
		}
	}
	for _, value := range []string{"PRIVATE_MAIN_ANSWER", "PRIVATE_MAIN_SUMMARY", "PRIVATE_THINKING", "PRIVATE_PRIOR_VERDICT", "HIDDEN_ARGUMENT"} {
		if strings.Contains(all.String(), value) {
			t.Fatal("untrusted or unrelated material included", value)
		}
	}
}

func TestLATMemoryKeepsOneLatestMemoAndRejectsStaleUpdates(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx, id := context.Background(), s.Session().ID
	for revision := 1; revision <= 20; revision++ {
		next := latmemory.State{Revision: revision, Through: int64(revision), Text: strings.Repeat("retained requirement ", 4000)}
		if err = s.InstallLATMemory(ctx, id, revision-1, next); err != nil {
			t.Fatal(err)
		}
	}
	var count, codec int
	if err = s.db.QueryRow("SELECT COUNT(*),MAX(codec) FROM lat_memory").Scan(&count, &codec); err != nil || count != 1 || codec != 2 {
		t.Fatal("memo retention or compression", count, codec, err)
	}
	before, err := s.LATMemory(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.InstallLATMemory(ctx, id, 0, latmemory.State{Revision: 1, Through: 21, Text: "stale overwrite"}); err == nil {
		t.Fatal("stale writer replaced the memo")
	}
	after, err := s.LATMemory(ctx, id)
	if err != nil || after != before {
		t.Fatal("stale update altered stored memory", err)
	}
	if _, err = s.db.Exec("UPDATE lat_memory SET body=X'00'"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.LATMemory(ctx, id); err == nil {
		t.Fatal("corrupt memo accepted")
	}
}
