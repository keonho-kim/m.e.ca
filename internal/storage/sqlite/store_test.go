// Purpose: Verify private session history survives database restart and supports bounded reads.
// Pattern: Integration tests using temporary storage.
// Usage: go test ./internal/storage/sqlite.
// Related: internal/storage/sqlite/store.go, internal/storage/sqlite/session.go, internal/storage/sqlite/runs.go.

package sqlite

import (
	"context"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"os"
	"path/filepath"
	"testing"
)

func TestSessionHistorySurvivesRestart(t *testing.T) {
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
	path := store.path
	if err := store.AppendMessage(ctx, session.ID, modelapi.Message{Role: "user", Content: "Fix auth"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
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
	restored := store.Session()
	if err != nil || restored.ID != session.ID {
		t.Fatalf("session: %+v, %v", restored, err)
	}
	history, err := store.Messages(ctx, session.ID, 100)
	if err != nil || len(history) != 1 || history[0].Content != "Fix auth" {
		t.Fatalf("history: %+v, %v", history, err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("database permissions: %v, %v", info, err)
	}
	other, err := catalog.Create(ctx, t.TempDir(), "other")
	if err != nil {
		t.Fatal(err)
	}
	if other.Session().ID == session.ID {
		t.Fatal("different projects shared one session")
	}
	if err := other.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestHistoryIsBounded(t *testing.T) {
	ctx := context.Background()
	store, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Error(err)
		}
	})
	session := store.Session()
	for _, text := range []string{"one", "two", "three"} {
		if err := store.AppendMessage(ctx, session.ID, modelapi.Message{Role: "user", Content: text}); err != nil {
			t.Fatal(err)
		}
	}
	history, err := store.Messages(ctx, session.ID, 2)
	if err != nil || len(history) != 2 || history[0].Content != "two" || history[1].Content != "three" {
		t.Fatalf("bounded history: %+v, %v", history, err)
	}
}
