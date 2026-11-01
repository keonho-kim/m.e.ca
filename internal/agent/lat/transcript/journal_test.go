// Purpose: Verify LAT transcripts isolate native history while retaining a durable parent-session audit.
// Pattern: SQLite integration tests.
// Usage: go test ./internal/agent/lat/transcript.
// Related: internal/agent/lat/transcript/journal.go, internal/storage/sqlite/events.go.
package transcript_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/lat/transcript"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/storage/sqlite"
)

func TestSideTranscriptIsolatesHistoryAndRecovery(t *testing.T) {
	ctx := context.Background()
	store := testStore(t)
	session := store.Session().ID
	main := modelapi.Message{Role: "user", Content: "MAIN_REQUEST"}
	checkpoint := checkpoint("parent_run")
	usage := modelapi.TokenUsage{Input: 2, Output: 3}
	if err := store.Commit(ctx, session, []runrecord.Entry{{Message: &main}, {Event: &runrecord.Event{Kind: "run_checkpoint", Checkpoint: &checkpoint}}, {Event: &runrecord.Event{Kind: "usage", Usage: &usage}}}); err != nil {
		t.Fatal(err)
	}
	side, err := transcript.New(store, session, checkpoint.ID, "plan")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(side.Close)
	request := modelapi.Message{Role: "user", Content: "SIDE_REQUEST"}
	call := toolapi.Call{ID: "read_1", Name: "read_file", Arguments: json.RawMessage(`{"path":"a.go"}`)}
	state := strings.Repeat("native state", 4000)
	response := modelapi.Message{Role: "assistant", ReasoningState: []byte(state), Calls: []toolapi.Call{call}}
	result := modelapi.Message{Role: "tool", ToolName: "read_file", ToolCallID: call.ID, Content: "SIDE_RESULT"}
	child := checkpoint
	child.ID, child.Status, child.Phase = "child_run", runrecord.Interrupted, "finished"
	child.Observation = "SIDE_INTERRUPTED"
	childUsage := modelapi.TokenUsage{Input: 20, Output: 30}
	statistics := runrecord.Statistics{ElapsedMillis: 10, GenerationMillis: 5, Usage: &childUsage}
	feedback := runrecord.Event{Kind: "lat_feedback", Text: "SIDE_FEEDBACK"}
	if err = side.Commit(ctx, session, []runrecord.Entry{{Message: &request}, {Message: &response}, {Message: &result}, {Event: &runrecord.Event{Kind: "run_checkpoint", RunID: child.ID, Checkpoint: &child, Statistics: &statistics}}, {Event: &runrecord.Event{Kind: "usage", Usage: &childUsage}}, {Event: &feedback}}); err != nil {
		t.Fatal(err)
	}
	messages, err := side.ContextMessages(ctx, session, 0, 1<<20, true)
	if err != nil || len(messages) != 3 || len(messages[1].Calls) != 1 || messages[1].Calls[0].ID != messages[2].ToolCallID || string(messages[1].ReasoningState) != state {
		t.Fatalf("native side history: %+v %v", messages, err)
	}
	mainMessages, err := store.ContextMessages(ctx, session, 0, 1<<20, true)
	if err != nil || len(mainMessages) != 1 || mainMessages[0].Content != main.Content {
		t.Fatalf("main context contaminated: %+v %v", mainMessages, err)
	}
	latest, err := store.LatestRun(ctx, session)
	if err != nil || latest == nil || latest.ID != checkpoint.ID || latest.Status != runrecord.Running {
		t.Fatalf("main checkpoint replaced: %+v %v", latest, err)
	}
	total, err := store.TokenUsage(ctx, session)
	if err != nil || total.Input != usage.Input+childUsage.Input || total.Output != usage.Output+childUsage.Output {
		t.Fatalf("actual side usage was not counted exactly once: %+v %v", total, err)
	}
	mark, err := store.MemoryWatermark(ctx, session, 0)
	if err != nil {
		t.Fatal(err)
	}
	records, _, err := store.MemoryRecords(ctx, session, 0, mark.Timeline)
	if err != nil || len(records) != 1 || records[0].Text != main.Content {
		t.Fatalf("main compaction contaminated: %+v %v", records, err)
	}
	visible, _, err := store.Timeline(ctx, session, 0, 100)
	if err != nil || len(visible) != 1 || visible[0].Kind != "message" {
		t.Fatalf("internal trace visible in parent history: %+v %v", visible, err)
	}
	latMark, err := store.LATWatermark(ctx, session, 0)
	if err != nil || latMark.Bytes != int64(len(main.Content)) {
		t.Fatalf("independent LAT memory contaminated: %+v %v", latMark, err)
	}
	var audit bytes.Buffer
	if err = store.Export(ctx, session, &audit); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"SIDE_REQUEST", "SIDE_RESULT", "SIDE_INTERRUPTED", "SIDE_FEEDBACK", `"lat_checkpoint"`, `"parent_run_id":"parent_run"`, `"phase":"plan"`, `"tool_call_id":"read_1"`, `"reasoning_state"`} {
		if !strings.Contains(audit.String(), want) {
			t.Errorf("audit missing %s", want)
		}
	}
	if !strings.Contains(audit.String(), base64.StdEncoding.EncodeToString(response.ReasoningState)) {
		t.Fatal("export lost externalized native continuation payload")
	}
	post, err := transcript.New(store, session, checkpoint.ID, "review")
	if err != nil {
		t.Fatal(err)
	}
	defer post.Close()
	if messages, err := post.ContextMessages(ctx, session, 0, 1<<20, true); err != nil || len(messages) != 0 {
		t.Fatalf("phases shared history: %+v %v", messages, err)
	}
}

func TestCancelledCommitLeavesSideHistoryUnchanged(t *testing.T) {
	store := testStore(t)
	session := store.Session().ID
	side, err := transcript.New(store, session, "parent_run", "review")
	if err != nil {
		t.Fatal(err)
	}
	defer side.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	m := modelapi.Message{Role: "assistant", Content: "CANCELLED_DRAFT", Incomplete: true}
	if err = side.Commit(ctx, session, []runrecord.Entry{{Message: &m}}); err == nil {
		t.Fatal("canceled commit succeeded")
	}
	if messages, err := side.ContextMessages(context.Background(), session, 0, 1<<20, true); err != nil || len(messages) != 0 {
		t.Fatalf("failed write changed local state: %+v %v", messages, err)
	}
	final := checkpoint("child_run")
	final.Status, final.Phase, final.Observation = runrecord.Interrupted, "finished", "cancelled"
	if err = side.Commit(context.WithoutCancel(ctx), session, []runrecord.Entry{{Message: &m}, {Event: &runrecord.Event{Kind: "run_checkpoint", RunID: final.ID, Checkpoint: &final}}}); err != nil {
		t.Fatal(err)
	}
	side.Close()
	if _, err = side.ContextMessages(context.Background(), session, 0, 1<<20, true); err == nil {
		t.Fatal("closed transcript still accepts reads")
	}
	var audit bytes.Buffer
	if err = store.Export(context.Background(), session, &audit); err != nil || !strings.Contains(audit.String(), "CANCELLED_DRAFT") || !strings.Contains(audit.String(), `"status":"interrupted"`) {
		t.Fatalf("cancellation audit lost: %s %v", audit.String(), err)
	}
	if latest, err := store.LatestRun(context.Background(), session); err != nil || latest != nil {
		t.Fatalf("child became recoverable main run: %+v %v", latest, err)
	}
}

func checkpoint(id string) runrecord.Checkpoint {
	return runrecord.Checkpoint{ID: id, Objective: "inspect", Phase: "decision", Status: runrecord.Running, InstructionsDigest: strings.Repeat("a", 64)}
}

func testStore(t *testing.T) *sqlite.Store {
	t.Helper()
	catalog, err := sqlite.OpenSessions(filepath.Join(t.TempDir(), "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	store, err := catalog.Create(context.Background(), root, root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Error(err)
		}
	})
	return store
}
