// Purpose: Verify selected paths respect scope and distinguish files from directories.
// Pattern: Filesystem behavior tests.
// Usage: go test ./internal/tools/local -run TestReferences.
// Related: internal/tools/local/references.go.
package local

import (
	"github.com/keonho-kim/v-code/internal/policy"
	"os"
	"path/filepath"
	"testing"
)

func TestReferences(t *testing.T) {
	e, root := testExecutor(t)
	if err := os.Mkdir(filepath.Join(root, "src"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "src/auth.go"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	settings := policy.Settings{Scope: policy.Directory}
	for _, path := range []string{"src/auth.go", "src"} {
		ref, err := e.Reference(settings, path)
		if err != nil || ref.Directory != (path == "src") {
			t.Fatalf("%q: %+v %v", path, ref, err)
		}
	}
	if _, err := e.Reference(settings, "../"); err == nil {
		t.Fatal("scope escape")
	}
}
