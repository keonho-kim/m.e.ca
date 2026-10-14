// Purpose: Verify native capability and serving-limit snapshots persist with exact model bindings.
// Pattern: Repository contract tests.
// Usage: go test ./internal/storage/sqlite -run TestNativeSelection.
// Related: internal/storage/sqlite/selection.go, internal/provider/connection/reasoning.go.
package sqlite

import (
	"context"
	"github.com/keonho-kim/v-code/internal/provider/connection"
	"testing"
)

func TestNativeSelection(t *testing.T) {
	store, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			t.Error(err)
		}
	}()
	ctx := context.Background()
	session := store.Session()
	conn, err := connection.New(connection.Ollama, "")
	if err != nil {
		t.Fatal(err)
	}
	on, level := true, "Deep"
	selected := connection.Selection{Connection: conn, Model: "alias", Thinking: &on, ReasoningEffort: &level, Details: &connection.ModelInfo{ID: "alias", ContextWindow: 8192, Reasoning: &connection.Reasoning{Levels: []string{"Light", "Deep"}, Default: "Light"}}}
	if err := store.SaveSelection(ctx, session.ID, selected); err != nil {
		t.Fatal(err)
	}
	restored, err := store.Selection(ctx, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if restored.Details.ContextWindow != 8192 || restored.Effort() != "Deep" || restored.ThinkingSupported {
		t.Fatalf("snapshot=%+v", restored)
	}
	off := false
	selected.Thinking = &off
	if err := store.SaveSelection(ctx, session.ID, selected); err == nil {
		t.Fatal("unsupported OFF persisted")
	}
	selected.Thinking = &on
	selected.Details.ID = "another-model"
	if err := store.SaveSelection(ctx, session.ID, selected); err == nil {
		t.Fatal("foreign model capabilities persisted")
	}
}
