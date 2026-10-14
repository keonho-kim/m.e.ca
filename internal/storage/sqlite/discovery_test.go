// Purpose: Verify filesystem discovery ordering, identity checks and missing-file behavior.
// Pattern: Repository boundary tests.
// Usage: go test ./internal/storage/sqlite -run 'TestDiscovery|TestMissingSession'.
// Related: internal/storage/sqlite/conversations.go, internal/storage/sqlite/catalog.go.
package sqlite

import (
	"context"
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoveryOrderRestoreAndRemoval(t *testing.T) {
	ctx := context.Background()
	catalog, err := OpenSessions(filepath.Join(t.TempDir(), "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	var ids []string
	for _, stamp := range []string{"2026-10-14T00:00:00Z", "2026-10-14T00:00:00.1Z", "2026-10-14T00:00:01Z"} {
		store, err := catalog.Create(ctx, root, root)
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, store.Session().ID)
		if _, err := store.db.ExecContext(ctx, "UPDATE sessions SET updated_at=?", stamp); err != nil {
			t.Fatal(err)
		}
		if _, err := store.db.ExecContext(ctx, "INSERT INTO sessions(id,project_root,display_root,created_at,updated_at) VALUES('another',?,?,?,?)", root, root, stamp, stamp); err == nil {
			t.Fatal("second session inserted into one DB")
		}
		if err := store.Close(); err != nil {
			t.Fatal(err)
		}
	}
	page, err := catalog.Conversations(ctx, 1, 1)
	if err != nil || len(page) != 1 || page[0].ID != ids[1] {
		t.Fatalf("page=%+v %v", page, err)
	}
	restored, err := catalog.Restore(ctx, root, root, true)
	if err != nil {
		t.Fatal(err)
	}
	if restored.Session().ID != ids[2] {
		t.Fatal("restore chose an older session")
	}
	if err := restored.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(catalog.directory, ids[2]+".db")); err != nil {
		t.Fatal(err)
	}
	list, err := catalog.Conversations(ctx, 0, 100)
	if err != nil || len(list) != 2 || list[0].ID != ids[1] {
		t.Fatalf("deleted file retained in registry: %+v %v", list, err)
	}
	for _, name := range []string{".pending-abandoned.db", ids[2] + ".db-wal", ids[2] + ".db-shm", ids[2] + ".db.lock", "v.db"} {
		if err := os.WriteFile(filepath.Join(catalog.directory, name), []byte("not a session"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	list, err = catalog.Conversations(ctx, 0, 100)
	if err != nil || len(list) != 2 {
		t.Fatalf("sidecar listed: %+v %v", list, err)
	}
}
func TestDiscoveryRejectsRenamedAndSymlinkedDatabase(t *testing.T) {
	for _, link := range []bool{false, true} {
		t.Run(map[bool]string{false: "renamed", true: "symlink"}[link], func(t *testing.T) {
			store, err := newTestStore(t)
			if err != nil {
				t.Fatal(err)
			}
			path := store.path
			catalog, err := OpenSessions(filepath.Dir(path))
			if err != nil {
				t.Fatal(err)
			}
			if err := store.Close(); err != nil {
				t.Fatal(err)
			}
			target := filepath.Join(catalog.directory, "session_"+rand.Text()+".db")
			if link {
				err = os.Symlink(path, target)
			} else {
				err = os.Rename(path, target)
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, err := catalog.Conversations(context.Background(), 0, 100); err == nil {
				t.Fatal("untrusted session accepted")
			}
		})
	}
}
func TestMissingSessionNeverCreatesDatabase(t *testing.T) {
	catalog, err := OpenSessions(filepath.Join(t.TempDir(), "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	id := "session_" + rand.Text()
	if _, err := catalog.Open(context.Background(), id); err == nil {
		t.Fatal("missing session opened")
	}
	if _, err := catalog.Open(context.Background(), "../other"); err == nil {
		t.Fatal("traversal accepted")
	}
	entries, err := os.ReadDir(catalog.directory)
	if err != nil || len(entries) != 0 {
		t.Fatalf("missing file recreated: %+v %v", entries, err)
	}
}
