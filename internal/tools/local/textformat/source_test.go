// Purpose: Preserve CRLF source observations while masking secrets and unsafe terminal controls.
// Pattern: Pure sanitization contract tests.
// Usage: go test ./internal/tools/local/textformat -run TestSourceRedaction.
// Related: internal/tools/local/textformat/source.go, internal/tools/local/command.go.
package textformat

import (
	"github.com/keonho-kim/v-code/internal/config/credentials"
	"strings"
	"testing"
)

func TestSourceRedactionPreservesOnlyUnmaskedCRLF(t *testing.T) {
	v := &credentials.Vault{}
	v.Remember("private-key")
	if got := RedactSource("a\r\nb\r\n", v.Redact); got != "a\r\nb\r\n" {
		t.Fatalf("CRLF source changed: %q", got)
	}
	if got := RedactSource("private-key\r\n", v.Redact); strings.Contains(got, "private-key") {
		t.Fatal("secret exposed")
	}
	if got := RedactSource("a\x1bb\r\n", v.Redact); strings.ContainsRune(got, '\x1b') {
		t.Fatal("terminal control exposed")
	}
}
