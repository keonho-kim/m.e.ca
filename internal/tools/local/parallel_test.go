// Purpose: Verify concurrent file operations exclude aliases and overlapping paths.
// Pattern: Filesystem scheduling contract tests.
// Usage: go test ./internal/tools/local -run TestParallelTargets.
// Related: internal/tools/local/parallel.go.
package local

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/policy"
)

func TestParallelTargets(t *testing.T) {
	e, root := testExecutor(t)
	if err := os.WriteFile(filepath.Join(root, "a"), []byte("same"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(filepath.Join(root, "a"), filepath.Join(root, "alias")); err != nil {
		t.Fatal(err)
	}
	settings := policy.Settings{Scope: policy.System}
	for _, tc := range []struct {
		a, b string
		want bool
	}{
		{"a", "b", true}, {"a", "a", false}, {"a", "alias", false}, {"dir", "dir/child", false},
	} {
		left := call("write_file", `{"path":"`+tc.a+`","content":"x"}`)
		right := call("write_file", `{"path":"`+tc.b+`","content":"y"}`)
		end, err := e.ParallelBatch(context.Background(), settings, []toolapi.Call{left, right}, 16)
		got := end == 2
		if err != nil || got != tc.want {
			t.Fatalf("%s/%s=%t %v", tc.a, tc.b, got, err)
		}
	}
}

func TestParallelLargeCandidatesUseSmallerWaves(t *testing.T) {
	e, root := testExecutor(t)
	var calls []toolapi.Call
	for _, name := range []string{"large-a", "large-b"} {
		path := filepath.Join(root, name)
		if err := os.WriteFile(path, nil, 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.Truncate(path, 16*1024*1024); err != nil {
			t.Fatal(err)
		}
		calls = append(calls, call("write_file", `{"path":"`+name+`","content":"body"}`))
	}
	end, err := e.ParallelBatch(context.Background(), policy.Settings{Scope: policy.System}, calls, 16)
	if err != nil || end != 1 {
		t.Fatalf("large candidates shared a preparation wave: %d %v", end, err)
	}
}
