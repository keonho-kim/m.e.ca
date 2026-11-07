// Purpose: Verify provider keys persist only as ciphertext and survive restart and saved-key replacement.
// Pattern: Credential security and lifecycle regression tests.
// Usage: go test ./internal/config/credentials -run TestEncrypted.
// Related: internal/config/credentials/keyring.go, internal/storage/credentialstore/store.go.
package credentials

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/keonho-kim/meca/internal/config/credentials/credentialtest"
)

func TestEncryptedCredentialRestartAndReplacement(t *testing.T) {
	keys := &credentialtest.Memory{}
	state := t.TempDir()
	v := &Vault{State: state, Keys: keys}
	const secret = "provider-api-key-private-fixture"
	if err := v.Set(t.Context(), "provider", "connection-scope", secret); err != nil {
		t.Fatal(err)
	}
	if _, err := keys.Get(t.Context(), "provider"); err == nil {
		t.Fatal("raw API key stored in OS keyring")
	}
	files, err := filepath.Glob(filepath.Join(state, "credentials.db*"))
	if err != nil || len(files) == 0 {
		t.Fatal("encrypted credential database missing", err)
	}
	for _, file := range files {
		body, err := os.ReadFile(file)
		if err != nil || bytes.Contains(body, []byte(secret)) {
			t.Fatal("plaintext key persisted", file, err)
		}
	}
	restarted := &Vault{State: state, Keys: keys}
	if got, err := restarted.Get(t.Context(), "provider", "connection-scope"); err != nil || got != secret {
		t.Fatal("credential lost across restart", err)
	}
	if err := restarted.Set(t.Context(), "provider", "connection-scope", secret+"-replaced"); err != nil {
		t.Fatal(err)
	}
	if got, err := v.Get(t.Context(), "provider", "connection-scope"); err != nil || got != secret+"-replaced" {
		t.Fatal("credential replacement lost", err)
	}
}
