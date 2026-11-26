// Purpose: Verify local encrypted credentials survive rotation, restarts and complete-directory backups.
// Pattern: Filesystem-backed vault integration tests.
// Usage: go test ./internal/config/credentials -run TestLocal.
// Related: internal/config/credentials/persistence.go, internal/storage/credentialkeys/store.go.
package credentials

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/keonho-kim/meca/internal/storage/credentialkeys"
	"github.com/keonho-kim/meca/internal/storage/credentialstore"
)

func TestLocalCredentialRotationAndBackup(t *testing.T) {
	v := &Vault{State: t.TempDir()}
	const original = "fixture-api-key-before"
	if err := v.Set(t.Context(), "provider", "scope", original); err != nil {
		t.Fatal(err)
	}
	backup, err := os.ReadFile(filepath.Join(v.State, credentialstore.Filename))
	if err != nil || bytes.Contains(backup, []byte(original)) {
		t.Fatal("plaintext API key in the credential database", err)
	}
	now := time.Now()
	if err := v.rotateOnce(t.Context(), now); err != nil {
		t.Fatal(err)
	}
	if err := v.rotateOnce(t.Context(), now.Add(rotationPeriod)); err != nil {
		t.Fatal(err)
	}
	if err := v.Set(t.Context(), "provider", "scope", "fixture-api-key-after"); err != nil {
		t.Fatal(err)
	}
	restarted := &Vault{State: v.State}
	if got, err := restarted.Get(t.Context(), "provider", "scope"); err != nil || got != "fixture-api-key-after" {
		t.Fatal("local credential did not survive restart", err)
	}
	entries, err := os.ReadDir(filepath.Join(v.State, credentialkeys.DirectoryName))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		body, err := os.ReadFile(filepath.Join(v.State, credentialkeys.DirectoryName, entry.Name()))
		if err != nil || bytes.Contains(body, []byte(original)) || bytes.Contains(body, []byte("fixture-api-key-after")) {
			t.Fatal("API key plaintext reached key files", err)
		}
	}
	restored := &Vault{State: t.TempDir()}
	restoredKeys := filepath.Join(restored.State, credentialkeys.DirectoryName)
	if err := os.Mkdir(restoredKeys, 0700); err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		body, err := os.ReadFile(filepath.Join(v.State, credentialkeys.DirectoryName, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(restoredKeys, entry.Name()), body, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(restored.State, credentialstore.Filename), backup, 0600); err != nil {
		t.Fatal(err)
	}
	if got, err := restored.Get(t.Context(), "provider", "scope"); err != nil || got != original {
		t.Fatal("local backup did not restore an older credential", err)
	}
}

func TestLocalMissingKeysPreserveCiphertext(t *testing.T) {
	v := &Vault{State: t.TempDir()}
	if err := v.Set(t.Context(), "provider", "scope", "fixture-api-key"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(v.State, credentialstore.Filename)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	keys := filepath.Join(v.State, credentialkeys.DirectoryName)
	if err := os.RemoveAll(keys); err != nil {
		t.Fatal(err)
	}
	if _, err := v.Get(t.Context(), "provider", "scope"); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("missing local keys were not reported", err)
	}
	if _, err := os.Stat(keys); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("reading a saved credential recreated its keys", err)
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("missing keys changed the database", err)
	}
}
