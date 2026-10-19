// Purpose: Keep original command captures once while excluding them from normal context and terminal projections.
// Pattern: Storage projection contract tests.
// Usage: go test ./internal/storage/sqlite -run TestCapturedOutput.
// Related: internal/storage/sqlite/events.go, internal/storage/sqlite/export.go.
package sqlite

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/keonho-kim/v-code/internal/agent/run"
	"strings"
	"testing"
)

func TestCapturedOutputIsDurableAndExcludedFromProjections(t *testing.T) {
	store, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	id := store.Session().ID
	capture, _ := json.Marshal(map[string]string{"Stdout": strings.Repeat("full-captured-output\n", 10000)})
	if err = store.RecordEvent(ctx, id, run.Event{Kind: "tool_succeeded", Capture: capture, Output: json.RawMessage(`{"Stdout":"excerpt"}`)}); err != nil {
		t.Fatal(err)
	}
	page, _, err := store.Timeline(ctx, id, 0, 10)
	if err != nil || len(page) != 1 || len(page[0].Event.Capture) != 0 {
		t.Fatalf("capture leaked into terminal projection: %v", err)
	}
	var exported bytes.Buffer
	if err = store.Export(ctx, id, &exported); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(exported.String(), "full-captured-output") {
		t.Fatal("original capture was not retained")
	}
}
