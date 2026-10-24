// Purpose: Verify partial edits reject source that changed before preparation.
// Pattern: Filesystem contract tests.
// Usage: go test ./internal/tools/local -run TestEditFailure.
// Related: internal/tools/local/prepare.go, internal/tools/local/textedit/edit.go.
package local

import (
	"context"
	"github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/policy"
	"os"
	"path/filepath"
	"testing"
)

func TestEditFailureReasons(t *testing.T) {
	executor, root := testExecutor(t)
	path := filepath.Join(root, "a")
	if err := os.WriteFile(path, []byte("first\nmiddle\nlast\n"), 0600); err != nil {
		t.Fatal(err)
	}
	edit := observedEdit(t, executor, "a", 1, 3, "fixed\n")
	if err := os.WriteFile(path, []byte("first\nCHANGED\nlast\n"), 0600); err != nil {
		t.Fatal(err)
	}
	settings := policy.Settings{Scope: policy.Directory}
	_, err := executor.Prepare(context.Background(), settings, edit)
	failure := tool.FailureOf(err)
	if failure == nil || failure.Code != "text_not_found" || failure.Next == "" {
		t.Fatalf("middle change accepted: %+v", failure)
	}
}
