// Purpose: Verify filename/path completion respects scope and finds directories.
// Pattern: Filesystem behavior tests.
// Usage: go test ./internal/tools/local -run TestReferences.
// Related: internal/tools/local/references.go.
package local

import (
	"context"
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
	settings := policy.Settings{Permission: policy.Approve, Scope: policy.Directory}
	for _, token := range []string{"auth", "src/au", "src"} {
		out, err := e.References(context.Background(), settings, token)
		if err != nil || len(out) == 0 {
			t.Fatalf("%q: %+v %v", token, out, err)
		}
	}
	if _, err := e.References(context.Background(), settings, "../"); err == nil {
		t.Fatal("scope escape")
	}
}
