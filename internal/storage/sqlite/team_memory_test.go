// Purpose: Preserve successfully authored public opinions through compaction using source references.
// Pattern: SQLite compaction integration tests.
// Usage: MemoryRecords resolves public records after native collaboration outcomes commit.
// Related: internal/storage/sqlite/team_memory.go, internal/storage/sqlite/memory_sources.go.
package sqlite

import (
	"context"
	"encoding/json"
	record "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/team/channel"
	"github.com/keonho-kim/meca/internal/agent/tool"
	"strings"
	"testing"
)

func TestAuthoredDiscussionCompactionUsesPublicOriginal(t *testing.T) {
	root, child := teamStorage(t)
	ctx := context.Background()
	const opinion = "Preserve the existing 401 response contract"
	message, err := root.AppendTeamMessage(ctx, channel.Message{From: "Hana", Text: opinion, Discussion: true})
	if err != nil {
		t.Fatal(err)
	}
	output, _ := json.Marshal(map[string]int64{"sequence": message.Sequence})
	event := record.Event{Kind: "tool_succeeded", CallID: "opinion", Call: &tool.Call{ID: "opinion", Name: "discuss", Arguments: json.RawMessage("{}")}, Operation: &tool.Operation{Name: "discuss", Status: "tool_succeeded", Result: "Published to the team"}, Output: output}
	if err = child.RecordEvent(ctx, child.session.ID, event); err != nil {
		t.Fatal(err)
	}
	var role, text string
	if err = child.db.QueryRow("SELECT memory_role,memory_text FROM tool_events").Scan(&role, &text); err != nil || text != "" || !strings.HasPrefix(role, "public:") {
		t.Fatal("public body duplicated in memory projection", role, text, err)
	}
	records, _, err := child.MemoryRecords(ctx, child.session.ID, 0, 1<<60)
	if err != nil || len(records) != 1 || records[0].Role != "peer" || !strings.Contains(records[0].Text, opinion) {
		t.Fatal(records, err)
	}
	if _, err = root.db.Exec("DELETE FROM tool_events"); err != nil {
		t.Fatal(err)
	}
	if _, _, err = child.MemoryRecords(ctx, child.session.ID, 0, 1<<60); err == nil {
		t.Fatal("missing authored original was silently omitted")
	}
}
