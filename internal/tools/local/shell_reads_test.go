// Purpose: Keep source inspection in shell tools and edit safety in frozen mutation candidates.
// Pattern: Tool boundary regression tests.
// Usage: go test ./internal/tools/local -run TestExactEdit.
// Related: internal/tools/local/catalog.go, internal/tools/local/candidate.go.
package local

import (
	"context"
	"github.com/keonho-kim/v-code/internal/policy"
	"os"
	"path/filepath"
	"testing"
)

func TestExactEditUsesCurrentSource(t *testing.T) {
	e, root := testExecutor(t)
	if err := os.WriteFile(filepath.Join(root, "a"), []byte("first\nsecond\n"), 0600); err != nil {
		t.Fatal(err)
	}
	op, err := e.Prepare(context.Background(), policy.Settings{Scope: policy.System, Permission: policy.AllowAll}, call("edit_file", `{"path":"a","edits":[{"old_text":"second","new_text":"changed"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = op.Execute(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(root, "a"))
	if err != nil || string(content) != "first\nchanged\n" {
		t.Fatalf("%q %v", content, err)
	}
}
