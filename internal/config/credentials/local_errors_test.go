// Purpose: Preserve actionable credential permission failures through the public vault boundary.
// Pattern: Filesystem-backed credential recovery test.
// Usage: go test ./internal/config/credentials -run TestLocalCredentialPermissionRecovery.
// Related: internal/config/credentials/persistence.go, internal/diagnostics/fault/error.go.
package credentials

import (
	"database/sql"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func TestLocalCredentialPermissionRecovery(t *testing.T) {
	v := &Vault{State: t.TempDir()}
	if err := v.Set(t.Context(), "provider", "scope", "fixture-api-key"); err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(v.State, "credential-keys")
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if err := os.Chmod(filepath.Join(directory, entry.Name()), 0644); err != nil {
			t.Fatal(err)
		}
	}
	_, err = v.Get(t.Context(), "provider", "scope")
	if fault.Code(err) != diag.FailedPrecondition {
		t.Fatalf("permission failure lost its cause: %v", err)
	}
	if message := fault.PublicMessage(err, v.Redact); !strings.Contains(message, "0600") {
		t.Fatalf("permission repair is not explained: %s", message)
	}
	if err := v.Set(t.Context(), "provider", "scope", "replacement-fixture"); fault.Code(err) != diag.FailedPrecondition {
		t.Fatalf("write permission failure lost its cause: %v", err)
	}
	for _, entry := range entries {
		if err := os.Chmod(filepath.Join(directory, entry.Name()), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if got, err := v.Get(t.Context(), "provider", "scope"); err != nil || got != "fixture-api-key" {
		t.Fatal("permission repair did not restore access", err)
	}
}

func TestLocalCredentialMissingKeyRecovery(t *testing.T) {
	v := &Vault{State: t.TempDir()}
	if err := v.Set(t.Context(), "provider", "scope", "fixture-api-key"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(v.State, "credential-keys", "keys.db")
	backup, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	uri := url.URL{Scheme: "file", Path: path, RawQuery: "mode=rw"}
	db, err := sql.Open("sqlite", uri.String())
	if err != nil {
		t.Fatal(err)
	}
	_, writeErr := db.ExecContext(t.Context(), "DELETE FROM keys")
	if err := errors.Join(writeErr, db.Close()); err != nil {
		t.Fatal(err)
	}
	_, err = v.Get(t.Context(), "provider", "scope")
	if !errors.Is(err, os.ErrNotExist) || fault.Code(err) != diag.NotFound {
		t.Fatalf("missing encryption key lost its diagnosis: %v", err)
	}
	if message := fault.PublicMessage(err, v.Redact); !strings.Contains(message, "backup") {
		t.Fatalf("missing encryption key did not explain recovery: %s", message)
	}
	if err := os.WriteFile(path, backup, 0600); err != nil {
		t.Fatal(err)
	}
	if got, err := v.Get(t.Context(), "provider", "scope"); err != nil || got != "fixture-api-key" {
		t.Fatal("key database restoration did not restore access", err)
	}
}
