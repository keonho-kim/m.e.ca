// Purpose: Preserve unrelated or malformed credential targets during token publication.
// Pattern: Credential filesystem failure contracts.
// Usage: go test ./internal/serverapi -run TestTokenRejectsInvalidTarget.
// Related: internal/serverapi/token.go, internal/config/statefs/files.go.
package serverapi

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTokenRejectsInvalidTarget(t *testing.T) {
	for _, body := range []string{"", "unrelated content", strings.Repeat("a", 257), strings.Repeat("a", 32) + "\nextra"} {
		path := filepath.Join(t.TempDir(), "token")
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
		if err := PublishToken(path, NewToken()); err == nil {
			t.Fatal("invalid existing target was overwritten")
		}
		after, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(after, []byte(body)) {
			t.Fatal("original invalid target changed", err)
		}
	}
}
