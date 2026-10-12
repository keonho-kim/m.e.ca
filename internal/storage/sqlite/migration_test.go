// Purpose: Verify prior schema upgrades and concurrent initialization preserve local state.
// Pattern: Repository lifecycle tests.
// Usage: go test ./internal/storage/sqlite -run 'TestVersionOne|TestConcurrentOpen'.
// Related: internal/storage/sqlite/schema.go, internal/storage/sqlite/store.go.
package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

func TestVersionOneMigration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(schema); err != nil {
		t.Fatal(err)
	}
	if err = db.Close(); err != nil {
		t.Fatal(err)
	}
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			t.Error(err)
		}
	}()
	var version int
	if err = store.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != 4 {
		t.Fatalf("schema: %d %v", version, err)
	}
	if _, _, err = store.Timeline(context.Background(), "missing", 0, 10); err != nil {
		t.Fatal(err)
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
