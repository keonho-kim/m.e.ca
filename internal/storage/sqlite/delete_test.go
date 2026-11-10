// Purpose: Verify conversation deletion cannot race a live writer or remove unrelated project content.
// Pattern: SQLite lease and tombstone integration tests.
// Usage: go test ./internal/storage/sqlite -run TestDeleteConversation.
// Related: internal/storage/sqlite/delete.go, internal/storage/sqlite/catalog.go.
package sqlite

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestDeleteConversationOwnsOnlyItsFiles(t *testing.T) {
	sessions, err := OpenSessions(filepath.Join(t.TempDir(), "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	project := t.TempDir()
	preserved := filepath.Join(project, "keep.txt")
	if err := os.WriteFile(preserved, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	first, err := sessions.Create(t.Context(), project, project)
	if err != nil {
		t.Fatal(err)
	}
	id := first.Session().ID
	directory := filepath.Dir(first.path)
	lease, err := first.LeaseSession()
	if err != nil {
		t.Fatal(err)
	}
	if err := sessions.DeleteConversation(t.Context(), project, id); err == nil {
		t.Fatal("deleted an open writer")
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	if err := lease.Close(); err != nil {
		t.Fatal(err)
	}
	team := filepath.Join(directory, ".team", id)
	if err := os.MkdirAll(team, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(team, "member.db"), []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := sessions.DeleteConversation(t.Context(), project, id); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{filepath.Join(directory, id+".db"), team} {
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("retained deleted files", path, err)
		}
	}
	if body, err := os.ReadFile(preserved); err != nil || string(body) != "keep" {
		t.Fatal("project content removed", err)
	}
	// Recreate an interrupted intent: startup cleanup finishes it without recreating a DB.
	marker := filepath.Join(directory, ".deleted", id)
	if err := os.MkdirAll(marker, 0700); err != nil {
		t.Fatal(err)
	}
	if err := sessions.CleanupDeleted(t.Context(), project); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("intent retained", err)
	}
	list, err := sessions.Conversations(t.Context(), project)
	if err != nil || len(list) != 0 {
		t.Fatal(list, err)
	}
}

func TestDeleteConversationCannotUnlinkAnOpenReadHandle(t *testing.T) {
	sessions, err := OpenSessions(filepath.Join(t.TempDir(), "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	project := t.TempDir()
	store, err := sessions.Create(t.Context(), project, project)
	if err != nil {
		t.Fatal(err)
	}
	id, path := store.Session().ID, store.path
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	reader, err := sessions.OpenRead(t.Context(), project, id)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	if err := sessions.DeleteConversation(t.Context(), project, id); err == nil {
		t.Fatal("open read handle was detached by deletion")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("read-protected database removed", err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(path), ".deleted", id)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("intent published before ownership", err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	if err := sessions.DeleteConversation(t.Context(), project, id); err != nil {
		t.Fatal(err)
	}
}
