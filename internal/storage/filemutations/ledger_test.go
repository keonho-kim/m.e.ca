// Purpose: Verify restart-safe effect admission, compact receipts and untouched unsupported formats.
// Pattern: SQLite persistence contract tests.
// Usage: go test -race ./internal/storage/filemutations.
// Related: internal/storage/filemutations/database.go, internal/storage/filemutations/ledger.go.
package filemutations

import (
	"bytes"
	"crypto/sha256"
	"os"
	"path/filepath"
	"testing"

	"github.com/keonho-kim/meca/internal/commands/server"
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func TestReceiptSurvivesRestartWithoutReadmittingEffects(t *testing.T) {
	directory := t.TempDir()
	ctx := t.Context()
	ledger, err := Open(ctx, directory)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256([]byte("intent"))
	value, duplicate, err := ledger.Admit(ctx, "session", "key-one-123", digest)
	if err != nil || duplicate || value.State != "pending" {
		t.Fatalf("admit: %+v %v %v", value, duplicate, err)
	}
	final := server.FileMutationResult{State: "applied", Source: "before", Path: "after", ReferencesReady: true, Paths: []string{"not-persisted"}}
	if err := ledger.Finish(ctx, "session", "key-one-123", final); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ledger.Admit(ctx, "session", "key-two-123", digest); err != nil {
		t.Fatal(err)
	}
	if err := ledger.Close(); err != nil {
		t.Fatal(err)
	}
	ledger, err = Open(ctx, directory)
	if err != nil {
		t.Fatal(err)
	}
	defer ledger.Close()
	value, duplicate, err = ledger.Admit(ctx, "session", "key-one-123", digest)
	if err != nil || !duplicate || value.State != "applied" || value.Path != "after" || value.Paths != nil {
		t.Fatalf("replay: %+v %v %v", value, duplicate, err)
	}
	pending, duplicate, err := ledger.Admit(ctx, "session", "key-two-123", digest)
	if err != nil || !duplicate || pending.State != "pending" {
		t.Fatalf("uncertain intent readmitted: %+v %v %v", pending, duplicate, err)
	}
	_, _, err = ledger.Admit(ctx, "session", "key-one-123", sha256.Sum256([]byte("different")))
	if fault.Code(err) != diag.Conflict {
		t.Fatalf("key conflict: %v", err)
	}
	var entries, charged, actual int
	if err := ledger.db.QueryRow("SELECT entries,bytes,(SELECT sum(charge) FROM receipts) FROM budget").Scan(&entries, &charged, &actual); err != nil {
		t.Fatal(err)
	}
	if entries != 2 || charged != actual || charged <= 16384 || charged >= 32768 {
		t.Fatalf("unbounded/inconsistent accounting: %d %d %d", entries, charged, actual)
	}
}
func TestUnsupportedLedgerIsPreserved(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "mutations.db")
	original := bytes.Repeat([]byte("preserve unsupported user data"), 8)
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	ledger, err := Open(t.Context(), directory)
	if ledger != nil {
		ledger.Close()
		t.Fatal("unsupported ledger opened")
	}
	if err == nil {
		t.Fatal("unsupported ledger accepted")
	}
	saved, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(saved, original) {
		t.Fatalf("unsupported database changed: %v", err)
	}
	for _, suffix := range []string{"-wal", "-shm", "-journal"} {
		if _, err := os.Stat(path + suffix); !os.IsNotExist(err) {
			t.Fatalf("unsupported database side effect: %s %v", suffix, err)
		}
	}
}
