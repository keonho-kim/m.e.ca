// Purpose: Verify provider-owned continuation bytes survive SQLite close and reopen unchanged.
// Pattern: Temporary-storage persistence contract test.
// Usage: go test ./internal/storage/sqlite -run TestProviderContinuation.
// Related: internal/storage/sqlite/messages.go, internal/agent/model/model.go.
package sqlite

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
)

func TestProviderContinuationSurvivesRestart(t *testing.T) {
	ctx := context.Background()
	catalog, err := OpenSessions(filepath.Join(t.TempDir(), "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	project := t.TempDir()
	store, err := catalog.Create(ctx, project, project)
	if err != nil {
		t.Fatal(err)
	}
	session := store.Session()
	state := []byte(`[{"type":"reasoning.encrypted","data":"opaque","format":"anthropic-claude-v1"}]`)
	err = store.AppendMessage(ctx, session.ID, modelapi.Message{Role: "assistant", Content: "done", ProviderID: "provider", ModelID: "model", ReasoningState: state})
	closeErr := store.Close()
	if err != nil || closeErr != nil {
		t.Fatalf("append=%v close=%v", err, closeErr)
	}
	store, err = catalog.Open(ctx, project, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Error(err)
		}
	})
	history, err := store.Messages(ctx, session.ID, 10)
	if err != nil || len(history) != 1 || !bytes.Equal(history[0].ReasoningState, state) || history[0].ProviderID != "provider" || history[0].ModelID != "model" {
		t.Fatalf("restored=%+v err=%v", history, err)
	}
}
