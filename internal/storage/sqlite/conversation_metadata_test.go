// Purpose: Verify unchanged metadata avoids new database handles and notices open-WAL writes.
// Pattern: SQLite metadata-index integration test.
// Usage: go test -race ./internal/storage/sqlite -run TestConversationMetadata.
// Related: internal/storage/sqlite/conversation_metadata.go.
package sqlite

import (
	"path/filepath"
	"testing"
)

func TestConversationMetadataReusesClosedDatabase(t *testing.T) {
	project := t.TempDir()
	catalog, err := OpenSessions(filepath.Join(t.TempDir(), "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	store, err := catalog.Create(t.Context(), project, project)
	if err != nil {
		t.Fatal(err)
	}
	file := conversationFile{path: store.path, id: store.Session().ID, group: filepath.Base(filepath.Dir(store.path))}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	first, err := openConversationReader(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	want, err := first.Read(t.Context(), file)
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	second, err := openConversationReader(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	got, err := second.Read(t.Context(), file)
	if err != nil || got != want {
		t.Fatal(got, want, err)
	}
	if second.db != nil {
		t.Fatal("unchanged metadata reopened SQLite")
	}
}

func TestConversationMetadataRechecksOpenWriterWAL(t *testing.T) {
	project := t.TempDir()
	catalog, err := OpenSessions(filepath.Join(t.TempDir(), "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	store, err := catalog.Create(t.Context(), project, project)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if _, err := store.db.ExecContext(t.Context(), "UPDATE sessions SET summary='before' WHERE singleton=1"); err != nil {
		t.Fatal(err)
	}
	first, _, err := catalog.ConversationPage(t.Context(), project, "", 50)
	if err != nil || len(first) != 1 || first[0].Summary != "before" {
		t.Fatal(first, err)
	}
	if _, err := store.db.ExecContext(t.Context(), "UPDATE sessions SET summary='after!' WHERE singleton=1"); err != nil {
		t.Fatal(err)
	}
	second, _, err := catalog.ConversationPage(t.Context(), project, "", 50)
	if err != nil || len(second) != 1 || second[0].Summary != "after!" {
		t.Fatal("stale WAL metadata", second, err)
	}
}
