// Purpose: Verify a native phase compacts privately while SQLite preserves its full audit and main memory.
// Pattern: SQLite integration tests.
// Usage: go test ./internal/agent/lat/transcript -run TestCompactionController.
// Related: internal/agent/lat/transcript/compaction.go, internal/agent/compact/controller.go.
package transcript_test

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/compact"
	"github.com/keonho-kim/meca/internal/agent/lat/transcript"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/prompt"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func TestCompactionControllerKeepsPhaseMemoryPrivate(t *testing.T) {
	ctx := context.Background()
	store := testStore(t)
	session := store.Session().ID
	main := modelapi.Message{Role: "user", Content: "MAIN_REQUEST"}
	if err := store.Commit(ctx, session, []runrecord.Entry{{Message: &main}}); err != nil {
		t.Fatal(err)
	}
	mainMemory := compact.Memory{Revision: 1, Through: 1, Markdown: "MAIN_SUMMARY"}
	if err := store.InstallMemory(ctx, session, 0, mainMemory); err != nil {
		t.Fatal(err)
	}
	if err := store.AddContextTokens(ctx, session, compact.TokenCount{Amount: 77, Kind: "request"}); err != nil {
		t.Fatal(err)
	}
	side, err := transcript.New(store, session, "parent_run", "review")
	if err != nil {
		t.Fatal(err)
	}
	defer side.Close()
	var messages []modelapi.Message
	messages = append(messages, modelapi.Message{Role: "user", Content: "SIDE_REQUEST"})
	for _, pair := range []struct{ id, content string }{{"old", strings.Repeat("OLDER_REVIEW_EVIDENCE\n", 2000)}, {"recent", "RECENT_EVIDENCE"}} {
		call := toolapi.Call{ID: pair.id, Name: "read_file", Arguments: json.RawMessage(`{"path":"source.go"}`)}
		messages = append(messages, modelapi.Message{Role: "assistant", Calls: []toolapi.Call{call}}, modelapi.Message{Role: "tool", ToolName: call.Name, ToolCallID: call.ID, Content: pair.content})
	}
	for i := range messages {
		if err = side.Commit(ctx, session, []runrecord.Entry{{Message: &messages[i]}}); err != nil {
			t.Fatal(err)
		}
	}
	settings := compact.Defaults()
	settings.KeepRecentTokens = 128
	controller := compact.Controller{Repository: side, Session: session, Prompt: prompt.Defaults().Compaction, Settings: settings, Redact: func(s string) string { return s }, ActiveRequest: "SIDE_REQUEST"}
	if err = controller.Observe(ctx, compact.TokenCount{Amount: 10000, Kind: "request"}); err != nil {
		t.Fatal(err)
	}
	calls := 0
	changed, err := controller.Compact(ctx, nil, func(_ context.Context, request modelapi.Request, emit func(modelapi.Event) error) error {
		calls++
		body, err := json.Marshal(request.Messages)
		if err != nil || !strings.Contains(string(body), "OLDER_REVIEW_EVIDENCE") || strings.Contains(string(body), "MAIN_REQUEST") || strings.Contains(string(body), "MAIN_SUMMARY") {
			t.Fatalf("private summary source: %s %v", body, err)
		}
		return emit(modelapi.Event{Delta: "SIDE_SUMMARY", Done: true})
	})
	if err != nil || !changed || calls != 1 {
		t.Fatalf("controller compaction: changed=%t calls=%d err=%v", changed, calls, err)
	}
	summary, boundary := controller.Context()
	tail, err := side.ContextMessages(ctx, session, boundary, 1024, true)
	if err != nil || summary != "SIDE_SUMMARY" || len(tail) != 3 || tail[0].Content != "SIDE_REQUEST" || tail[1].Calls[0].ID != tail[2].ToolCallID {
		t.Fatalf("compacted private context: summary=%s tail=%+v err=%v", summary, tail, err)
	}
	restored, err := store.Memory(ctx, session)
	if err != nil || restored != mainMemory {
		t.Fatalf("main memory replaced: %+v %v", restored, err)
	}
	mark, err := store.MemoryWatermark(ctx, session, 0)
	if err != nil || mark.Tokens != 77 || mark.Usage != 1 {
		t.Fatalf("main token samples changed: %+v %v", mark, err)
	}
	var audit bytes.Buffer
	if err = store.Export(ctx, session, &audit); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"kind":"lat_compaction"`, "SIDE_SUMMARY", "OLDER_REVIEW_EVIDENCE", `"phase":"review"`} {
		if !strings.Contains(audit.String(), want) {
			t.Errorf("full audit missing %s", want)
		}
	}
}
