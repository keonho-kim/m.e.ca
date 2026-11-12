// Purpose: Include public argument metadata in tool-original byte admission.
// Pattern: Bounded SQLite projection regression.
// Usage: go test ./internal/storage/sqlite -run TestToolDetailBudgetIncludesMetadata.
// Related: internal/storage/sqlite/tool_detail.go, internal/tui/session/tool_details.go.
package sqlite

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/storage/history"
)

func TestToolDetailBudgetIncludesMetadata(t *testing.T) {
	store, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	arguments, err := json.Marshal(map[string]string{"command": strings.Repeat("x", 128<<10)})
	if err != nil {
		t.Fatal(err)
	}
	event := record.Event{Kind: "tool_succeeded", CallID: "large", Call: &tool.Call{ID: "large", Name: "execute_command", Arguments: arguments}, Output: []byte(`{"Status":"exited","ExitCode":0}`)}
	if err := store.RecordEvent(t.Context(), store.Session().ID, event); err != nil {
		t.Fatal(err)
	}
	_, err = store.ToolEvent(t.Context(), record.EventReference{Session: store.Session().ID, Sequence: 1}, history.ReadBounds{MaxBytes: 64 << 10})
	if !errors.Is(err, history.ErrReadLimit) {
		t.Fatal("large public arguments escaped original-read admission", err)
	}
}
