// Purpose: Verify partial edits reject stale, malformed or unavailable read anchors.
// Pattern: Filesystem contract tests.
// Usage: go test ./internal/tools/local -run TestEditFailure.
// Related: internal/tools/local/prepare.go, internal/tools/local/anchors/edit.go.
package local

import (
	"context"
	"github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/policy"
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
	edit := anchoredEdit(t, executor, "a", 1, 3, "fixed\n")
	if err := os.WriteFile(path, []byte("first\nCHANGED\nlast\n"), 0600); err != nil {
		t.Fatal(err)
	}
	settings := policy.Settings{Permission: policy.AllowAll, Scope: policy.Directory}
	_, err := executor.Prepare(context.Background(), settings, edit)
	failure := tool.FailureOf(err)
	if failure == nil || failure.Code != tool.StaleTarget || failure.Next == "" {
		t.Fatalf("middle change accepted: %+v", failure)
	}
	executor.reads.Clear()
	_, err = executor.Prepare(context.Background(), settings, edit)
	failure = tool.FailureOf(err)
	if failure == nil || failure.Code != tool.AnchorUnavailable {
		t.Fatalf("expired read accepted: %+v", failure)
	}
	if _, err := executor.Prepare(context.Background(), settings, call("edit_file", `{"path":"a","old_text":"first","new_text":"fixed"}`)); err == nil {
		t.Fatal("obsolete edit format accepted")
	}
}
