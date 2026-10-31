// Purpose: Verify canonical review/state storage avoids repeated bodies while preserving recovery and export.
// Pattern: Persistence integration tests.
// Usage: go test ./internal/storage/sqlite -run TestCanonical.
// Related: internal/storage/sqlite/run_state.go, internal/storage/sqlite/context.go, internal/storage/sqlite/export.go.
package sqlite

import (
	"bytes"
	"context"
	"encoding/json"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"strings"
	"testing"
)

func TestCanonicalRunStateReplacesSnapshots(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx, id := context.Background(), s.Session().ID
	cp := runrecord.Checkpoint{ID: "run_test", Objective: "work", Status: runrecord.Running, Phase: "decision", InstructionsDigest: strings.Repeat("a", 64)}
	for i := range 100 {
		cp.ModelCalls = i
		cp.Work.Files = append(cp.Work.Files, toolapi.FileFact{Path: strings.Repeat("file", 100), Version: "v1"})
		if err = s.RecordEvent(ctx, id, runrecord.Event{Kind: "run_checkpoint", RunID: cp.ID, Checkpoint: &cp}); err != nil {
			t.Fatal(err)
		}
	}
	var states, events int
	s.db.QueryRow("SELECT COUNT(*) FROM run_states").Scan(&states)
	s.db.QueryRow("SELECT COUNT(*) FROM tool_events").Scan(&events)
	if states != 1 || events != 0 {
		t.Fatalf("intermediate snapshots retained: states=%d events=%d", states, events)
	}
	var codec, raw, stored int
	if err = s.db.QueryRow("SELECT codec,raw_bytes,length(body) FROM run_states").Scan(&codec, &raw, &stored); err != nil || codec != 2 || stored >= raw {
		t.Fatalf("structured state was not compressed: codec=%d raw=%d stored=%d error=%v", codec, raw, stored, err)
	}
	got, err := s.LatestRun(ctx, id)
	if err != nil || got.ModelCalls != 99 || len(got.Work.Files) != 100 {
		t.Fatalf("state lost: %+v %v", got, err)
	}
	cp.Phase = "finished"
	cp.Status = runrecord.Answered
	if err = s.RecordEvent(ctx, id, runrecord.Event{Kind: "run_checkpoint", RunID: cp.ID, Checkpoint: &cp}); err != nil {
		t.Fatal(err)
	}
	var exported bytes.Buffer
	if err = s.Export(ctx, id, &exported); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(exported.String(), strings.Repeat("file", 100)) {
		t.Fatal("export lost canonical file facts")
	}
}

func TestCanonicalReviewRestoresEveryProjection(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx, id := context.Background(), s.Session().ID
	text := "required correction " + strings.Repeat("detail ", 10000)
	if err = s.RecordEvent(ctx, id, runrecord.Event{Kind: "lat_feedback", Text: text}); err != nil {
		t.Fatal(err)
	}
	var stored string
	if err = s.db.QueryRow("SELECT memory_text FROM tool_events").Scan(&stored); err != nil || stored != "" {
		t.Fatal("review source duplicated into memory_text", err)
	}
	messages, err := s.ContextMessages(ctx, id, 0, 1<<20, false)
	if err != nil || len(messages) != 1 || !strings.Contains(messages[0].Content, text) {
		t.Fatal("restored review context missing", err)
	}
	mark, err := s.MemoryWatermark(ctx, id, 0)
	if err != nil || mark.Tokens == 0 {
		t.Fatal("review feedback omitted from context occupancy", err)
	}
	records, _, err := s.MemoryRecords(ctx, id, 0, mark.Timeline)
	if err != nil || len(records) != 1 || records[0].Text != text {
		t.Fatal("compaction lost review source", err)
	}
	var body []byte
	s.db.QueryRow("SELECT body FROM tool_events").Scan(&body)
	var fields map[string]json.RawMessage
	if json.Unmarshal(body, &fields) != nil {
		t.Fatal("invalid canonical body")
	}
}

func TestCanonicalRunStateOrderingAndRollback(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx, id := context.Background(), s.Session().ID
	cp := runrecord.Checkpoint{ID: "first", Objective: "work", Status: runrecord.Running, Phase: "decision", InstructionsDigest: strings.Repeat("a", 64)}
	for _, run := range []string{"first", "second", "first"} {
		cp.ID = run
		if err = s.RecordEvent(ctx, id, runrecord.Event{Kind: "run_checkpoint", Checkpoint: &cp}); err != nil {
			t.Fatal(err)
		}
	}
	got, err := s.LatestRun(ctx, id)
	if err != nil || got.ID != "first" {
		t.Fatal("latest state did not follow committed updates", err)
	}
	cp.ModelCalls = 42
	updated := runrecord.Event{Kind: "run_checkpoint", Checkpoint: &cp}
	invalid := runrecord.Event{Kind: "usage"}
	if err = s.Commit(ctx, id, []runrecord.Entry{{Event: &updated}, {Event: &invalid}}); err == nil {
		t.Fatal("invalid journal group committed")
	}
	got, err = s.LatestRun(ctx, id)
	if err != nil || got.ModelCalls != 0 || got.ID != "first" {
		t.Fatal("state update escaped the failed transaction", err)
	}
}
