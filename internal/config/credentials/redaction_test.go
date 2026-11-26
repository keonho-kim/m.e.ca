// Purpose: Verify known credentials and partial streaming prefixes stay private.
// Pattern: Security behavior tests.
// Usage: go test ./internal/config/credentials.
// Related: internal/config/credentials/persistence.go.
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
	if strings.Contains(v.NewStreamMask()("start provider-sec", true), "provider-sec") {
		t.Fatal("streamed secret prefix")
	}
}
