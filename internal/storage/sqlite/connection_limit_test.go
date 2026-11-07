// Purpose: Keep the page ceiling on replacement connections and preserve rows after a full transaction.
// Pattern: SQLite resource-limit lifecycle test.
// Usage: go test ./internal/storage/sqlite -run TestConnectionPageLimit.
// Related: internal/storage/sqlite/connection_limit.go, internal/storage/sqlite/store.go.
package sqlite

import (
	"strconv"
	"testing"
)

func TestConnectionPageLimitSurvivesReplacement(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	s.db.SetMaxIdleConns(0)
	for range 2 {
		var maximum, pageBytes int64
		if err := s.db.QueryRowContext(t.Context(), "PRAGMA max_page_count").Scan(&maximum); err != nil {
			t.Fatal(err)
		}
		if err := s.db.QueryRowContext(t.Context(), "PRAGMA page_size").Scan(&pageBytes); err != nil {
			t.Fatal(err)
		}
		if maximum*pageBytes != maxSessionBytes {
			t.Fatal("replacement connection lost capacity ceiling", maximum, pageBytes)
		}
	}
	s.db.SetMaxIdleConns(1)
	if _, err := s.db.ExecContext(t.Context(), "CREATE TABLE capacity_probe(body BLOB)"); err != nil {
		t.Fatal(err)
	}
	var pages int64
	if err := s.db.QueryRowContext(t.Context(), "PRAGMA page_count").Scan(&pages); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(t.Context(), "PRAGMA max_page_count="+strconv.FormatInt(pages+1, 10)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(t.Context(), "INSERT INTO capacity_probe VALUES(zeroblob(1048576))"); err == nil {
		t.Fatal("transaction exceeded its page limit")
	}
	var count int
	if err := s.db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM capacity_probe").Scan(&count); err != nil || count != 0 {
		t.Fatal("failed allocation left partial state", count, err)
	}
}
