// Purpose: Verify known credentials are removed and partial streaming prefixes stay private.
// Pattern: Security behavior tests.
// Usage: go test ./internal/config/credentials.
// Related: internal/config/credentials/keyring.go.
package credentials

import (
	"github.com/zalando/go-keyring"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	if strings.Contains(v.Stream("start provider-sec"), "provider-sec") {
		t.Fatal("streamed secret prefix")
	}
}

func TestClearDeletesOnlyVServiceAndMarksSavedCredentials(t *testing.T) {
	keyring.MockInit()
	v := &Vault{State: t.TempDir()}
	if err := v.Set("provider", "provider-secret"); err != nil {
		t.Fatal(err)
	}
	if err := keyring.Set("other-app", "provider", "other-secret"); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(v.State, "credentials.used"))
	if err != nil || strings.Contains(string(body), "provider-secret") {
		t.Fatal("credential cleanup marker missing or contains a secret")
	}
	if err := Clear(); err != nil {
		t.Fatal(err)
	}
	if _, err := keyring.Get(service, "provider"); err == nil {
		t.Fatal("V credential remains")
	}
	if value, err := keyring.Get("other-app", "provider"); err != nil || value != "other-secret" {
		t.Fatal("other service was modified")
	}
}
