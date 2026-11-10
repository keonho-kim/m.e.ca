// Purpose: Verify task titles read only requested metadata, preserve project isolation and notice WAL commits.
// Pattern: Scoped SQLite metadata integration contracts.
// Usage: go test ./internal/storage/sqlite -run TestConversationSummaries.
// Related: internal/storage/sqlite/conversation_titles.go, internal/storage/sqlite/conversation_reader.go.
package sqlite

import (
	"context"
	"crypto/rand"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/model"
)

func TestConversationSummariesStayScopedAndNoticeWAL(t *testing.T) {
	catalog, err := OpenSessions(filepath.Join(t.TempDir(), "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	project, foreignProject := t.TempDir(), t.TempDir()
	create := func(root, title string) (*Store, string) {
		t.Helper()
		store, err := catalog.Create(t.Context(), root, root)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := store.Close(); err != nil {
				t.Error(err)
			}
		})
		if title != "" {
			if err := store.AppendMessage(t.Context(), store.Session().ID, model.Message{Role: "user", Content: title}); err != nil {
				t.Fatal(err)
			}
		}
		return store, store.Session().ID
	}
	first, firstID := create(project, "")
	_, secondID := create(project, "Second conversation")
	_, foreignID := create(foreignProject, "Private foreign title")
	if _, err := catalog.ConversationSummaries(t.Context(), project, []string{firstID}); err != nil {
		t.Fatal(err)
	}
	if err := first.AppendMessage(t.Context(), firstID, model.Message{Role: "user", Content: "한국어 작업 제목"}); err != nil {
		t.Fatal(err)
	}
	brokenID := "session_" + rand.Text()
	brokenPath := filepath.Join(filepath.Dir(first.path), brokenID+".db")
	if err := os.WriteFile(brokenPath, []byte("not a database"), 0600); err != nil {
		t.Fatal(err)
	}
	missing := "session_" + rand.Text()
	values, err := catalog.ConversationSummaries(t.Context(), project, []string{firstID, secondID, firstID, foreignID, missing})
	if err != nil || len(values) != 2 || values[firstID] != "한국어 작업 제목" || values[secondID] != "Second conversation" {
		t.Fatalf("scoped metadata: %#v, %v", values, err)
	}
	if _, err := os.Lstat(filepath.Join(filepath.Dir(first.path), missing+".db.guard")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("missing metadata lookup created a guard file", err)
	}
	if _, err := catalog.ConversationSummaries(t.Context(), project, []string{brokenID}); err == nil {
		t.Fatal("corrupt requested metadata was accepted")
	}
	bytes, err := os.ReadFile(brokenPath)
	if err != nil || string(bytes) != "not a database" {
		t.Fatal("metadata read changed original storage", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := catalog.ConversationSummaries(ctx, project, []string{firstID}); !errors.Is(err, context.Canceled) {
		t.Fatal("canceled metadata lookup", err)
	}
	if _, err := catalog.ConversationSummaries(t.Context(), project, []string{"../invalid"}); err == nil {
		t.Fatal("invalid identity accepted")
	}
	if _, err := catalog.ConversationSummaries(t.Context(), project, make([]string, 101)); err == nil {
		t.Fatal("unbounded metadata lookup accepted")
	}
}

func TestConversationSummariesReleaseDeletionLeases(t *testing.T) {
	project := t.TempDir()
	catalog, err := OpenSessions(filepath.Join(t.TempDir(), "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	store, err := catalog.Create(t.Context(), project, project)
	if err != nil {
		t.Fatal(err)
	}
	id := store.Session().ID
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.ConversationSummaries(t.Context(), project, []string{id}); err != nil {
		t.Fatal(err)
	}
	if err := catalog.DeleteConversation(t.Context(), project, id); err != nil {
		t.Fatal("metadata lookup retained its deletion lease", err)
	}
	values, err := catalog.ConversationSummaries(t.Context(), project, []string{id})
	if err != nil || len(values) != 0 {
		t.Fatal("deleted conversation retained in title results", values, err)
	}
}
