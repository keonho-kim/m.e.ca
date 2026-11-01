// Purpose: Verify side sessions persist canonical evidence without repeated state or schema growth.
// Pattern: SQLite storage-contract regression.
// Usage: Run TestLATStorageAvoidsRepeatedBookkeeping.
// Related: internal/agent/lat/transcript/audit.go, internal/storage/sqlite/events.go.
package sqlite

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/lat/transcript"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func TestLATStorageAvoidsRepeatedBookkeeping(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx, id := context.Background(), s.Session().ID
	tableCount := func() int {
		var count int
		if err := s.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table'").Scan(&count); err != nil {
			t.Fatal(err)
		}
		return count
	}
	beforeTables := tableCount()
	j, err := transcript.New(s, id, "parent_run", "review")
	if err != nil {
		t.Fatal(err)
	}
	defer j.Close()
	request := modelapi.Message{Role: "user", Content: "MANDATORY_REQUEST\n" + strings.Repeat("original evidence\n", 4096)}
	if err := j.Commit(ctx, id, []runrecord.Entry{{Message: &request}}); err != nil {
		t.Fatal(err)
	}
	call := toolapi.Call{ID: "read_1", Name: "execute_command", Arguments: json.RawMessage(`{"command":"cat source.go"}`)}
	assistant := modelapi.Message{Role: "assistant", Calls: []toolapi.Call{call}}
	if err := j.Commit(ctx, id, []runrecord.Entry{{Message: &assistant}}); err != nil {
		t.Fatal(err)
	}
	cp := runrecord.Checkpoint{ID: "child_run", Objective: request.Content, Status: runrecord.Running, Phase: "decision", InstructionsDigest: strings.Repeat("a", 64)}
	for step := range 100 {
		cp.ModelCalls = step
		cp.Work.Files = append(cp.Work.Files, toolapi.FileFact{Path: "SNAPSHOT_ONLY_" + strings.Repeat("source", 100), Version: "version"})
		entries := []runrecord.Entry{
			{Event: &runrecord.Event{Kind: "run_checkpoint", RunID: cp.ID, Checkpoint: &cp}},
			{Event: &runrecord.Event{Kind: "tool_proposed", CallID: call.ID, Call: &call}},
			{Event: &runrecord.Event{Kind: "tool_started", CallID: call.ID, Call: &call}},
		}
		if err := j.Commit(ctx, id, entries); err != nil {
			t.Fatal(err)
		}
	}
	result := modelapi.Message{Role: "tool", ToolCallID: call.ID, ToolName: call.Name, Content: `{"Failure":{"Detail":"READ_FAILURE"}}`}
	usage := modelapi.TokenUsage{Input: 20, Output: 5}
	cp.Phase, cp.Status = "finished", runrecord.Interrupted
	cp.Observation = "Interrupted after the read failure"
	if err := j.Commit(ctx, id, []runrecord.Entry{
		{Message: &result},
		{Event: &runrecord.Event{Kind: "tool_failed", CallID: call.ID, Call: &call, Text: "READ_FAILURE"}},
		{Event: &runrecord.Event{Kind: "usage", Usage: &usage}},
		{Event: &runrecord.Event{Kind: "run_checkpoint", RunID: cp.ID, Checkpoint: &cp}},
	}); err != nil {
		t.Fatal(err)
	}
	var checkpoints, transient, displays, snapshots int
	if err := s.db.QueryRow(`SELECT
 SUM(kind='lat_checkpoint'),SUM(kind IN ('tool_proposed','tool_started')),
 SUM(CAST(display_args AS TEXT)<>'{}'),SUM(kind='lat_checkpoint' AND raw_bytes>4096)
 FROM tool_events`).Scan(&checkpoints, &transient, &displays, &snapshots); err != nil {
		t.Fatal(err)
	}
	if checkpoints != 1 || transient != 0 || displays != 0 || snapshots != 0 {
		t.Fatalf("redundant bookkeeping: checkpoints=%d transient=%d displays=%d snapshots=%d", checkpoints, transient, displays, snapshots)
	}
	for _, table := range []string{"run_states", "memories", "conversation_tokens", "lat_memory"} {
		var count int
		if err := s.db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("side session extended %s: %d rows", table, count)
		}
	}
	if tableCount() != beforeTables {
		t.Fatal("side session extended the schema")
	}
	var exported bytes.Buffer
	if err := s.Export(ctx, id, &exported); err != nil {
		t.Fatal(err)
	}
	audit := exported.String()
	if strings.Count(audit, "MANDATORY_REQUEST") != 1 || strings.Contains(audit, "SNAPSHOT_ONLY_") {
		t.Fatal("native request or state snapshots duplicated in durable audit")
	}
	for _, value := range []string{"READ_FAILURE", "Interrupted after the read failure", `"tool_call_id":"read_1"`, `"input":20`} {
		if !strings.Contains(audit, value) {
			t.Fatalf("audit lost %q", value)
		}
	}
	if total, err := s.TokenUsage(ctx, id); err != nil || total.Input != 20 || total.Output != 5 {
		t.Fatalf("usage lost or duplicated: %+v %v", total, err)
	}
}
