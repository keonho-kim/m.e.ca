// Purpose: Verify phase-private compaction retains native groups and publishes only durable summaries.
// Pattern: Repository boundary tests.
// Usage: go test ./internal/agent/lat/transcript.
// Related: internal/agent/lat/transcript/compaction.go, internal/storage/history/boundary.go.
package transcript

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/compact"
	"github.com/keonho-kim/meca/internal/agent/media"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func TestPrivateMemoryPreservesRequestAndRecentNativeGroup(t *testing.T) {
	ctx := context.Background()
	parent := &failingCommitter{}
	journal, err := New(parent, "session", "parent_run", "plan")
	if err != nil {
		t.Fatal(err)
	}
	defer journal.Close()
	commitMessage(t, journal, modelapi.Message{Role: "user", Content: "LATEST_REQUEST"})
	for _, pair := range []struct{ id, output string }{{"old", strings.Repeat("large listing\n", 3000)}, {"recent", "current observation"}} {
		call := toolapi.Call{ID: pair.id, Name: "read_file", Arguments: json.RawMessage(`{"path":"source.go"}`)}
		commitMessage(t, journal, modelapi.Message{Role: "assistant", Calls: []toolapi.Call{call}, ReasoningState: []byte(pair.id)})
		commitMessage(t, journal, modelapi.Message{Role: "tool", ToolName: call.Name, ToolCallID: call.ID, Content: pair.output})
	}
	if err = journal.AddContextTokens(ctx, "session", compact.TokenCount{Amount: 10000, Kind: "request"}); err != nil {
		t.Fatal(err)
	}
	mark, err := journal.MemoryWatermark(ctx, "session", 0)
	if err != nil || mark.Tokens != 10000 || mark.Usage != 1 || mark.Timeline != 5 {
		t.Fatalf("private watermark: %+v %v", mark, err)
	}
	boundary, err := journal.CompactionBoundary(ctx, "session", mark.Timeline, 1024)
	if err != nil || boundary != 3 {
		t.Fatalf("native group split: boundary=%d err=%v", boundary, err)
	}
	if smallest, err := journal.CompactionBoundary(ctx, "session", mark.Timeline, 0); err != nil || smallest != boundary {
		t.Fatalf("small tail separated native calls/results: boundary=%d err=%v", smallest, err)
	}
	records, cursor, err := journal.MemoryRecords(ctx, "session", 0, boundary)
	if err != nil || cursor != boundary || len(records) != 2 || records[0].Text != "LATEST_REQUEST" || !strings.Contains(records[1].Text, "large listing") {
		t.Fatalf("private source projection: %+v cursor=%d err=%v", records, cursor, err)
	}
	next := compact.Memory{Revision: 1, Through: boundary, UsageThrough: mark.Usage, Markdown: "Earlier listing observed"}
	parent.err = errors.New("audit unavailable")
	if err = journal.InstallMemory(ctx, "session", 0, next); !errors.Is(err, parent.err) || journal.memory.Revision != 0 || len(journal.messages) != 5 {
		t.Fatalf("failed audit changed memory: %+v %v", journal.memory, err)
	}
	parent.err = nil
	if err = journal.InstallMemory(ctx, "session", 0, next); err != nil {
		t.Fatal(err)
	}
	if len(journal.messages) != 3 || journal.memory != next {
		t.Fatalf("summarized history retained: count=%d memory=%+v", len(journal.messages), journal.memory)
	}
	messages, err := journal.ContextMessages(ctx, "session", boundary, 1024, true)
	if err != nil || len(messages) != 3 || messages[0].Content != "LATEST_REQUEST" || len(messages[1].Calls) != 1 || messages[1].Calls[0].ID != messages[2].ToolCallID || string(messages[1].ReasoningState) != "recent" {
		t.Fatalf("request or recent native pair lost: %+v %v", messages, err)
	}
	if _, err = journal.ContextMessages(ctx, "session", 0, 1<<20, true); err == nil {
		t.Fatal("read before pruned memory boundary accepted")
	}
	mark, err = journal.MemoryWatermark(ctx, "session", next.UsageThrough)
	if err != nil || mark.Tokens != compact.EstimateBytes(int64(len("current observation"))).Amount || mark.Usage != next.UsageThrough {
		t.Fatalf("compacted token sample reused: %+v %v", mark, err)
	}
	if err = journal.InstallMemory(ctx, "session", 0, next); err == nil {
		t.Fatal("stale revision installed")
	}
	commitMessage(t, journal, modelapi.Message{Role: "user", Content: "NEW_REQUEST"})
	if len(journal.messages) != 3 || journal.messages[0].message.Role != "assistant" {
		t.Fatal("superseded summarized request reference retained")
	}
	journal.Close()
	if journal.memory != (compact.Memory{}) || journal.sample != (contextSample{}) || journal.sequence != 0 || journal.latestUser != 0 {
		t.Fatal("Close retained private compaction state")
	}
}

func TestPrivateCompactionRejectsCanceledMutation(t *testing.T) {
	journal, err := New(&failingCommitter{}, "session", "parent_run", "review")
	if err != nil {
		t.Fatal(err)
	}
	defer journal.Close()
	commitMessage(t, journal, modelapi.Message{Role: "user", Content: "request"})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err = journal.AddContextTokens(ctx, "session", compact.TokenCount{Amount: 20, Kind: "request"}); !errors.Is(err, context.Canceled) || journal.sample.sequence != 0 {
		t.Fatalf("cancelled sample mutation: %+v %v", journal.sample, err)
	}
	if err = journal.InstallMemory(ctx, "session", 0, compact.Memory{Revision: 1, Through: 1, Markdown: "summary"}); !errors.Is(err, context.Canceled) || journal.memory.Revision != 0 {
		t.Fatalf("cancelled memory publication: %+v %v", journal.memory, err)
	}
}

func TestPrivateCompactionRetainsUnreadImageGroup(t *testing.T) {
	journal, err := New(&failingCommitter{}, "session", "parent_run", "review")
	if err != nil {
		t.Fatal(err)
	}
	defer journal.Close()
	image, err := media.New("source.png", "image/png", "png", 1, 1, []byte{1})
	if err != nil {
		t.Fatal(err)
	}
	call := toolapi.Call{ID: "image", Name: "read_image", Arguments: json.RawMessage(`{"path":"source.png"}`)}
	for _, message := range []modelapi.Message{{Role: "user", Content: "review"}, {Role: "assistant", Calls: []toolapi.Call{call}}, {Role: "tool", ToolName: call.Name, ToolCallID: call.ID, Images: []media.Image{image}}, {Role: "user", Content: "inspect the visible text"}} {
		commitMessage(t, journal, message)
	}
	if boundary, err := journal.CompactionBoundary(context.Background(), "session", journal.sequence, 0); err != nil || boundary != 1 {
		t.Fatalf("unread image group could be summarized: boundary=%d err=%v", boundary, err)
	}
	commitMessage(t, journal, modelapi.Message{Role: "assistant", Content: "image consumed"})
	if boundary, err := journal.CompactionBoundary(context.Background(), "session", journal.sequence, 0); err != nil || boundary != 4 {
		t.Fatalf("consumed image remained pinned: boundary=%d err=%v", boundary, err)
	}
}

func commitMessage(t *testing.T, journal *Journal, message modelapi.Message) {
	t.Helper()
	if err := journal.Commit(context.Background(), "session", []runrecord.Entry{{Message: &message}}); err != nil {
		t.Fatal(err)
	}
}
