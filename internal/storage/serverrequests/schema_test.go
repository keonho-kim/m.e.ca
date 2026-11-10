// Purpose: Preserve prior request databases when the native-only format cannot open them.
// Pattern: SQLite format-boundary and file-preservation regression test.
// Usage: go test ./internal/storage/serverrequests -run TestPreviousRequestSchema.
// Related: internal/storage/serverrequests/database.go.
package serverrequests

import (
	"bytes"
	"database/sql"
	"github.com/keonho-kim/meca/internal/commands/server"
	"os"
	"testing"
)

func TestPreviousRequestSchemaIsRejectedWithoutRecreation(t *testing.T) {
	directory, scope := t.TempDir(), server.ProjectScope("fixture")
	ledger, err := Open(t.Context(), directory, scope)
	if err != nil {
		t.Fatal(err)
	}
	path := ledger.path
	if _, _, err := ledger.Admit(t.Context(), "session_AAAAAAAAAAAAAAAAAAAAAAAAAA", "original", server.InputDigest("preserved", nil)); err != nil {
		t.Fatal(err)
	}
	if err := ledger.Close(); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("PRAGMA user_version=3"); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if reopened, err := Open(t.Context(), directory, scope); err == nil || reopened != nil {
		if reopened != nil {
			_ = reopened.Close()
		}
		t.Fatal("unsupported database was silently reused")
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("unsupported database was recreated or changed", err)
	}
}
