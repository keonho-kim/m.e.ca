// Purpose: Verify unsupported session databases are preserved and concurrent creation never publishes partial files.
// Pattern: Repository lifecycle tests.
// Usage: go test ./internal/storage/sqlite -run 'TestUnsupportedSchema|TestConcurrentSession'.
// Related: internal/storage/sqlite/catalog.go, internal/storage/sqlite/store.go, internal/storage/sqlite/schema.go.
package sqlite

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestUnsupportedSchemaPreserved(t *testing.T) {
	for _, version := range []int{0, 1, 2, 3, 4, 6} {
		t.Run(fmt.Sprint(version), func(t *testing.T) {
			catalog, err := OpenSessions(filepath.Join(t.TempDir(), "sessions"))
			if err != nil {
				t.Fatal(err)
			}
			group, project, err := catalog.projectDirectory(t.TempDir(), true)
			if err != nil {
				t.Fatal(err)
			}
			id := "session_" + rand.Text()
			path := filepath.Join(group, id+".db")
			db, err := sql.Open("sqlite", path)
			if err != nil {
				t.Fatal(err)
			}
			_, setupErr := db.Exec(fmt.Sprintf("CREATE TABLE archived(body TEXT); INSERT INTO archived VALUES ('keep'); PRAGMA user_version=%d;", version))
			closeErr := db.Close()
			if setupErr != nil || closeErr != nil {
				t.Fatalf("fixture: %v %v", setupErr, closeErr)
			}
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			store, openErr := catalog.Open(context.Background(), project, id)
			if store != nil {
				if err := store.Close(); err != nil {
					t.Error(err)
				}
			}
			if openErr == nil {
				t.Fatal("unsupported schema accepted")
			}
			if _, err := catalog.Conversations(context.Background(), 0, 100); err == nil {
				t.Fatal("unsupported DB silently hidden from discovery")
			}
			after, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(before, after) {
				t.Fatal("unsupported database was modified")
			}
		})
	}
}
func TestConcurrentSessionCreation(t *testing.T) {
	catalog, err := OpenSessions(filepath.Join(t.TempDir(), "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	results := make(chan error, 8)
	for n := 0; n < 8; n++ {
		go func() {
			store, err := catalog.Create(context.Background(), root, root)
			if err == nil {
				err = store.Close()
			}
			if err == nil {
				_, err = catalog.Conversations(context.Background(), 0, 100)
			}
			results <- err
		}()
	}
	for n := 0; n < 8; n++ {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
	list, err := catalog.Conversations(context.Background(), 0, 100)
	if err != nil || len(list) != 8 {
		t.Fatalf("partial/duplicate session publication: %+v %v", list, err)
	}
}
