// Purpose: Verify known credentials are removed and partial streaming prefixes stay private.
// Pattern: Security behavior tests.
// Usage: go test ./internal/config/credentials.
// Related: internal/config/credentials/keyring.go.
package credentials

import (
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
