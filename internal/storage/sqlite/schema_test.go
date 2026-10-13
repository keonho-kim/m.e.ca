// Purpose: Verify unsupported schemas are preserved and current initialization is serialized.
// Pattern: Repository lifecycle tests.
// Usage: go test ./internal/storage/sqlite -run 'TestUnsupportedSchema|TestConcurrentOpen'.
// Related: internal/storage/sqlite/schema.go, internal/storage/sqlite/store.go.
package sqlite

import (
	"bytes"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestUnsupportedSchemaPreserved(t *testing.T) {
	for _, version := range []int{0, 1, 2, 3, 5} {
		t.Run(fmt.Sprint(version), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "v.db")
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
			store, openErr := Open(path)
			if store != nil {
				if err := store.Close(); err != nil {
					t.Error(err)
				}
			}
			if openErr == nil {
				t.Fatal("unsupported schema accepted")
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
func TestConcurrentOpen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v.db")
	results := make(chan error, 4)
	for n := 0; n < 4; n++ {
		go func() {
			store, err := Open(path)
			if err == nil {
				err = store.Close()
			}
			results <- err
		}()
	}
	for n := 0; n < 4; n++ {
		if err := <-results; err != nil {
			t.Error(err)
		}
	}
}
