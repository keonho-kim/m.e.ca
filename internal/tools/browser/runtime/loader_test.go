// Purpose: Check actionable loader errors reach the command without exposing surrounding stderr.
// Pattern: Public diagnostic contract regression.
// Usage: go test ./internal/tools/browser/runtime -run TestMissingLibrary.
// Related: internal/tools/browser/runtime/loader.go, internal/tools/browser/runtime/install.go.
package runtime

import (
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func TestMissingLibrary(t *testing.T) {
	stderr := []byte("private value: chrome: error while loading shared libraries: libnss3.so: cannot open shared object file: No such file or directory")
	err := MissingLibrary(stderr)
	if err == nil {
		t.Fatal("loader failure lost")
	}
	public := fault.PublicMessage(preparationFailure(err), func(text string) string { return text })
	if !strings.Contains(public, "libnss3.so") || strings.Contains(public, "private value") {
		t.Fatal(public)
	}
	if MissingLibrary([]byte("ordinary helper error")) != nil {
		t.Fatal("wrong error classification")
	}
}
