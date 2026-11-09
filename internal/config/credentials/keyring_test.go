// Purpose: Verify known credentials are removed and partial streaming prefixes stay private.
// Pattern: Security behavior tests.
// Usage: go test ./internal/config/credentials.
// Related: internal/config/credentials/keyring.go.
package credentials

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/config/credentials/credentialtest"
)

func TestRedactionAndStreaming(t *testing.T) {
	v := &Vault{}
	v.Remember("provider-secret")
	if v.Redact("the provider-secret key\x1b[31m") != "the [redacted] key[31m" {
		t.Fatal("secret or terminal escape leaked")
	}
	if v.Redact("provide") != "provide" {
		t.Fatal("ordinary text was redacted")
	}
	if strings.Contains(v.NewStreamMask()("start provider-sec", true), "provider-sec") {
		t.Fatal("streamed secret prefix")
	}
}

func TestClearDeletesOnlyMECAServiceAndMarksSavedCredentials(t *testing.T) {
	keys := &credentialtest.Memory{}
	v := &Vault{State: t.TempDir(), Keys: keys}
	if err := v.Set(t.Context(), "provider", "connection-scope", "provider-secret"); err != nil {
		t.Fatal(err)
	}
	if value, err := v.Get(t.Context(), "provider", "connection-scope"); err != nil || value != "provider-secret" {
		t.Fatal("encrypted credential was not readable", err)
	}

	body, err := os.ReadFile(filepath.Join(v.State, "credentials.used"))
	if err != nil || string(body) != "meca\n" || strings.Contains(string(body), "provider-secret") {
		t.Fatal("credential cleanup marker missing or contains a secret")
	}
	if err := v.Clear(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err := keys.Get(t.Context(), "provider"); err == nil {
		t.Fatal("MECA credential remains")
	}
	if _, err := v.Get(t.Context(), "provider", "connection-scope"); err == nil {
		t.Fatal("MECA encryption key remains after uninstall cleanup")
	}

}
