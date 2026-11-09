// Purpose: Store one immutable original for a tool message and its matching event.
// Pattern: SQLite storage contract regression test.
// Usage: go test ./internal/storage/sqlite -run TestToolPayloadSharesOneOriginal.
// Related: internal/storage/content/pack.go, internal/storage/sqlite/prepared_payload.go.
package sqlite

import (
	"encoding/base64"
	"encoding/json"
	"math/rand"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func TestToolPayloadSharesOneOriginal(t *testing.T) {
	store, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	input := make([]byte, 64<<10)
	if _, err := rand.New(rand.NewSource(1)).Read(input); err != nil {
		t.Fatal(err)
	}
	output, err := json.Marshal(map[string]string{"body": base64.StdEncoding.EncodeToString(input)})
	if err != nil {
		t.Fatal(err)
	}
	call := toolapi.Call{ID: "call_one", Name: "load_skill", Arguments: json.RawMessage(`{"id":"project:review"}`)}
	message := modelapi.Message{Role: "tool", ToolName: call.Name, ToolCallID: call.ID, Content: string(output)}
	event := runrecord.Event{Kind: "tool_succeeded", CallID: call.ID, Call: &call, Output: output}
	if err := store.Commit(t.Context(), store.Session().ID, []runrecord.Entry{{Message: &message}, {Event: &event}}); err != nil {
		t.Fatal(err)
	}
	var count, raw, stored int
	if err := store.db.QueryRowContext(t.Context(), "SELECT COUNT(*),SUM(raw_bytes),SUM(length(body)) FROM payloads").Scan(&count, &raw, &stored); err != nil {
		t.Fatal(err)
	}
	t.Logf("single tool result %d bytes: %d distinct payloads, %d raw bytes, %d stored bytes", len(output), count, raw, stored)
	if count != 1 {
		t.Fatalf("same semantic tool output stored in %d distinct payload representations", count)
	}
}
