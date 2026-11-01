// Purpose: Keep current path and project checks independent of earlier scheduling snapshots.
// Pattern: Filesystem substitution regression tests.
// Usage: Run TestPlanningBoundary with the native local tools.
// Related: internal/tools/local/parallel.go, internal/tools/local/directories.go.
package local

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/policy"
)

func TestPlanningBoundaryRechecksSymlinksBeforePreparation(t *testing.T) {
	e, project := testExecutor(t)
	ordinary := t.TempDir()
	alias := filepath.Join(project, "alias")
	if err := os.Symlink(ordinary, alias); err != nil {
		t.Fatal(err)
	}
	settings := policy.Settings{Scope: policy.System}
	candidate := call("write_file", `{"path":"alias/file","content":"new"}`)
	if plan, err := e.ParallelBatch(context.Background(), settings, []toolapi.Call{candidate}, 16); err != nil || plan.Count != 1 {
		t.Fatal(plan, err)
	}
	if err := os.Remove(alias); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(e.home, ".ssh"), alias); err != nil {
		t.Fatal(err)
	}
	// Use an existing private target so this checks protected-path resolution,
	// rather than merely rejecting a dangling symlink.
	if err := os.MkdirAll(filepath.Join(e.home, ".ssh"), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Prepare(context.Background(), settings, candidate); err == nil {
		t.Fatal("a scheduling snapshot bypassed the private-path guard")
	}
}

func TestPlanningBoundaryRechecksProjectBeforePreparation(t *testing.T) {
	e, project := testExecutor(t)
	settings := policy.Settings{Scope: policy.System}
	candidate := call("create_directory", `{"paths":["new/a","new/b"]}`)
	if _, err := e.ParallelBatch(context.Background(), settings, []toolapi.Call{candidate}, 16); err != nil {
		t.Fatal(err)
	}
	old := project + ".old"
	if err := os.Rename(project, old); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(project); err != nil {
			t.Error(err)
		}
		if err := os.Rename(old, project); err != nil {
			t.Error(err)
		}
	})
	if err := os.Mkdir(project, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Prepare(context.Background(), settings, candidate); err == nil {
		t.Fatal("a scheduling snapshot accepted a different project root")
	}
	control := call("list_command_runs", `{}`)
	if plan, err := e.ParallelBatch(context.Background(), settings, []toolapi.Call{control}, 16); err != nil || plan.Count != 1 {
		t.Fatal("project replacement blocked owned process management", plan, err)
	}
}

func BenchmarkDirectoryPreparation(b *testing.B) {
	project, home := b.TempDir(), b.TempDir()
	e, err := NewExecutor(project, home, filepath.Join(home, ".meca"), func(s string) string { return s })
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { e.Close() })
	paths := ""
	for i := range 16 {
		if i > 0 {
			paths += ","
		}
		paths += fmt.Sprintf(`"dir%d/nested/leaf"`, i)
	}
	candidate := call("create_directory", `{"paths":[`+paths+`]}`)
	b.ReportAllocs()
	for b.Loop() {
		if _, err := e.Prepare(context.Background(), policy.Settings{Scope: policy.System}, candidate); err != nil {
			b.Fatal(err)
		}
	}
}
