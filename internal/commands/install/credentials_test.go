// Purpose: Verify installation prepares private local credentials and retains them on reinstall.
// Pattern: Filesystem installation integration tests.
// Usage: go test ./internal/commands/install -run TestInstallLocalCredentials.
// Related: internal/commands/install/install.go, internal/config/credentials/persistence.go.
package install

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/keonho-kim/meca/internal/config/credentials"
)

func TestInstallLocalCredentials(t *testing.T) {
	o := options(t, "/bin/sh")
	if _, err := Install(t.Context(), o); err != nil {
		t.Fatal(err)
	}
	state := filepath.Join(o.Home, ".meca")
	keys := filepath.Join(state, "credential-keys")
	info, err := os.Stat(keys)
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0700 {
		t.Fatalf("installation did not prepare private local keys: %v", err)
	}
	v := &credentials.Vault{State: state}
	if err := v.Set(t.Context(), "provider", "scope", "fixture-api-key"); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(t.Context(), o); err != nil {
		t.Fatal(err)
	}
	restarted := &credentials.Vault{State: state}
	if got, err := restarted.Get(t.Context(), "provider", "scope"); err != nil || got != "fixture-api-key" {
		t.Fatal("reinstall lost the credential", err)
	}
}

func TestInstallRejectsUnsafeCredentialDirectory(t *testing.T) {
	o := options(t, "/bin/sh")
	state := filepath.Join(o.Home, ".meca")
	if err := os.Mkdir(state, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(t.TempDir(), filepath.Join(state, "credential-keys")); err != nil {
		t.Fatal(err)
	}
	if result, err := Install(t.Context(), o); err == nil || result.Binary != "" {
		t.Fatal("installation accepted a redirected credential directory", err)
	}
	if _, err := os.Stat(BinaryPath(o.Home)); !os.IsNotExist(err) {
		t.Fatal("installation published a binary after credential preparation failed", err)
	}
}
