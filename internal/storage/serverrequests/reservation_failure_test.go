// Purpose: Return known unpublished admission capacity while keeping successful deduplication durable.
// Pattern: SQLite failure-boundary integration test.
// Usage: go test -race ./internal/storage/serverrequests -run TestAdmissionInsertFailure.
// Related: internal/storage/serverrequests/ledger.go, internal/storage/capacity/reservation.go.
package serverrequests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/keonho-kim/meca/internal/commands/server"
)

func TestAdmissionInsertFailureReturnsUnusedCapacity(t *testing.T) {
	directory := t.TempDir()
	l, err := Open(t.Context(), directory, server.ProjectScope("fixture"))
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	if _, err := l.db.Exec("CREATE TRIGGER reject_admission BEFORE INSERT ON requests BEGIN SELECT RAISE(ABORT,'fixture insertion failed'); END"); err != nil {
		t.Fatal(err)
	}
	const session = "session_AAAAAAAAAAAAAAAAAAAAAAAAAA"
	if _, _, err := l.Admit(t.Context(), session, "key", server.InputDigest("input", nil)); err == nil {
		t.Fatal("failed insertion accepted")
	}
	body, err := os.ReadFile(filepath.Join(directory, "capacity", "allocated"))
	if err != nil || string(body) != "v1\n0\n" {
		t.Fatal("unused reservation retained", string(body), err)
	}
	if _, err := l.db.Exec("DROP TRIGGER reject_admission"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := l.Admit(t.Context(), session, "key", server.InputDigest("input", nil)); err != nil {
		t.Fatal(err)
	}
	if _, duplicate, err := l.Admit(t.Context(), session, "key", server.InputDigest("input", nil)); err != nil || !duplicate {
		t.Fatal("deduplication lost", duplicate, err)
	}
	body, err = os.ReadFile(filepath.Join(directory, "capacity", "allocated"))
	if err != nil || string(body) != "v1\n1\n" {
		t.Fatal("successful admission consumed incorrect capacity", string(body), err)
	}
}
