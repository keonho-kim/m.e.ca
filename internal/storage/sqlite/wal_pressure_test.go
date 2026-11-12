// Purpose: Stop WAL growth behind a pinned reader before accepting another write.
// Pattern: Actual SQLite resource-pressure regression.
// Usage: go test ./internal/storage/sqlite -run TestWALPressure.
// Related: internal/storage/sqlite/write.go, internal/storage/sqlite/store.go.
package sqlite

import (
	"database/sql"
	"github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	"testing"
)

func TestWALPressureRejectsBeforeEffectsAndRecovers(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err := s.db.Exec("CREATE TABLE pressure_fixture(body BLOB)"); err != nil {
		t.Fatal(err)
	}
	reader, err := connect(t.Context(), s.path, true)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	tx, err := reader.BeginTx(t.Context(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	var count int
	if err := tx.QueryRow("SELECT count(*) FROM pressure_fixture").Scan(&count); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 18; i++ {
		if _, err := s.db.Exec("INSERT INTO pressure_fixture VALUES(zeroblob(1048576))"); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.SelectFile(t.Context(), s.session.ID, "blocked", true); fault.Code(err) != catalog.ResourceExhausted {
		t.Fatal("write escaped WAL pressure", err)
	}
	var rows int
	if err := s.db.QueryRow("SELECT count(*) FROM file_selections WHERE path='blocked'").Scan(&rows); err != nil || rows != 0 {
		t.Fatal("rejected write had effects", rows, err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if err := s.SelectFile(t.Context(), s.session.ID, "recovered", true); err != nil {
		t.Fatal("released reader did not restore writes", err)
	}
}
