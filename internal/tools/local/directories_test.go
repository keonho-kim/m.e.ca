// Purpose: Verify recursive multi-directory creation and preparation safety.
// Pattern: Filesystem contract tests.
// Usage: go test ./internal/tools/local -run TestDirectories.
// Related: internal/tools/local/directories.go.
package local

import (
	"context"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/policy"
	"os"
	"path/filepath"
	"testing"
)

func TestDirectories(t *testing.T) {
	e, root := testExecutor(t)
	ctx := context.Background()
	settings := policy.Settings{Scope: policy.Directory}
	c := call("create_directory", `{"paths":["a/b/c","a/d","a/b/c"]}`)
	for round := 0; round < 2; round++ {
		op, err := e.Prepare(ctx, settings, c)
		if err != nil {
			t.Fatal(err)
		}
		result, err := op.Execute(ctx, true)
		if err != nil {
			t.Fatal(err)
		}
		want := 4
		if round == 1 {
			want = 0
		}
		if len(result.Changes) != want {
			t.Fatalf("changes: %+v", result)
		}
	}
	for _, p := range []string{"a/b/c", "a/d"} {
		info, err := os.Stat(filepath.Join(root, p))
		if err != nil || !info.IsDir() {
			t.Fatalf("%s: %v", p, err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "file"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Prepare(ctx, settings, call("create_directory", `{"paths":["uncreated","file/child"]}`)); err == nil {
		t.Fatal("accepted file parent")
	}
	if _, err := os.Stat(filepath.Join(root, "uncreated")); !os.IsNotExist(err) {
		t.Fatal("preflight mutated filesystem")
	}
	end, err := e.ParallelBatch(ctx, settings, []toolapi.Call{call("create_directory", `{"paths":["shared/a"]}`), call("create_directory", `{"paths":["shared/b"]}`)}, 16)
	if err != nil || end != 1 {
		t.Fatalf("shared missing parent: %v %v", end, err)
	}
}

func TestDirectoriesStaleParent(t *testing.T) {
	e, root := testExecutor(t)
	ctx := context.Background()
	settings := policy.Settings{Scope: policy.Directory}
	op, err := e.Prepare(ctx, settings, call("create_directory", `{"paths":["a/b"]}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "other"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("other", filepath.Join(root, "a")); err != nil {
		t.Fatal(err)
	}
	if _, err := op.Execute(ctx, true); err == nil {
		t.Fatal("accepted replaced parent")
	}
	if _, err := os.Stat(filepath.Join(root, "other/b")); !os.IsNotExist(err) {
		t.Fatal("created redirected directory")
	}
}

func TestDirectoriesRejectWholeInvalidRequest(t *testing.T) {
	e, root := testExecutor(t)
	ctx := context.Background()
	settings := policy.Settings{Scope: policy.Directory}
	for _, args := range []string{
		`{"paths":["valid","../outside"]}`,
		`{"paths":["valid","bad\npath"]}`,
		`{"paths":[]}`,
	} {
		if _, err := e.Prepare(ctx, settings, call("create_directory", args)); err == nil {
			t.Fatalf("accepted %s", args)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "valid")); !os.IsNotExist(err) {
		t.Fatal("invalid request created a directory")
	}
}
