// Purpose: Verify encrypted rotation and old-backup recovery against disposable real Keychain entries.
// Pattern: Opt-in native credential persistence integration test.
// Usage: MECA_TEST_KEYCHAIN=1 go test ./internal/config/credentials -run TestNativeEncryptedVault.
// Related: internal/config/credentials/keyring.go, internal/config/credentials/osstore/store_darwin.go.
package credentials

import (
	"bytes"
	"context"
	"crypto/rand"
	"github.com/keonho-kim/meca/internal/config/credentials/osstore"
	"github.com/keonho-kim/meca/internal/storage/credentialstore"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNativeEncryptedVaultRotationAndBackup(t *testing.T) {
	if os.Getenv("MECA_TEST_KEYCHAIN") != "1" {
		t.Skip("requires explicit native Keychain verification")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	keys := osstore.Store{Service: "meca-vault-test-" + rand.Text()}
	defer func() {
		cleanup, stop := context.WithTimeout(context.WithoutCancel(t.Context()), 10*time.Second)
		defer stop()
		if err := keys.Clear(cleanup); err != nil {
			t.Errorf("remove disposable service %s: %v", keys.Service, err)
		}
	}()
	v := &Vault{State: t.TempDir(), Keys: keys}
	if err := v.Set(ctx, "provider", "scope", "fixture-api-key-before"); err != nil {
		t.Fatal(err)
	}
	backup, err := os.ReadFile(filepath.Join(v.State, credentialstore.Filename))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(backup, []byte("fixture-api-key-before")) {
		t.Fatal("plaintext API key in database")
	}
	now := time.Now()
	if err := v.rotateOnce(ctx, now); err != nil {
		t.Fatal(err)
	}
	if err := v.rotateOnce(ctx, now.Add(rotationPeriod)); err != nil {
		t.Fatal("native periodic rotation failed", err)
	}
	rotated, err := os.ReadFile(filepath.Join(v.State, credentialstore.Filename))
	if err != nil || bytes.Equal(backup, rotated) {
		t.Fatal("periodic rotation did not change ciphertext", err)
	}
	if got, err := v.Get(ctx, "provider", "scope"); err != nil || got != "fixture-api-key-before" {
		t.Fatal("periodic rotation changed API key", err)
	}
	if err := v.Set(ctx, "provider", "scope", "fixture-api-key-after"); err != nil {
		t.Fatal(err)
	}
	if got, err := v.Get(ctx, "provider", "scope"); err != nil || got != "fixture-api-key-after" {
		t.Fatal("native rotation lost current key", err)
	}
	restored := &Vault{State: t.TempDir(), Keys: keys}
	if err := os.WriteFile(filepath.Join(restored.State, credentialstore.Filename), backup, 0600); err != nil {
		t.Fatal(err)
	}
	if got, err := restored.Get(ctx, "provider", "scope"); err != nil || got != "fixture-api-key-before" {
		t.Fatal("old backup lost after native rotation", err)
	}
	if err := keys.Clear(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := restored.Get(ctx, "provider", "scope"); err == nil {
		t.Fatal("missing OS key unexpectedly recovered")
	}
	after, err := os.ReadFile(filepath.Join(restored.State, credentialstore.Filename))
	if err != nil || !bytes.Equal(after, backup) {
		t.Fatal("missing key overwrote original ciphertext", err)
	}
}
