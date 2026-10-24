// Purpose: Verify dedicated file tools canonicalize host paths, guard private state and pin the project.
// Pattern: Filesystem policy contract tests.
// Usage: go test ./internal/tools/local/filesystem.
// Related: internal/tools/local/filesystem/boundary.go, internal/tools/local/filesystem/paths.go.
package filesystem

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBoundary(t *testing.T) {
	project, home := t.TempDir(), t.TempDir()
	b, err := New(project, home, filepath.Join(home, ".meca"))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{filepath.Join(project, "new"), filepath.Join(home, "ordinary-file")} {
		if _, err := b.ReadTarget(path); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := b.ReadTarget(filepath.Join(home, ".ssh/key")); err == nil {
		t.Fatal("dedicated file tool exposed a protected path")
	}
	alias := filepath.Join(project, "linked-home")
	if err := os.Symlink(home, alias); err != nil {
		t.Fatal(err)
	}
	if _, err := b.ReadTarget(filepath.Join(alias, ".meca/config.toml")); err == nil {
		t.Fatal("symlink bypassed private-path guard")
	}
	if err := b.Movable(b.Project()); err == nil {
		t.Fatal("project root was movable")
	}
	if err := os.Rename(project, project+".old"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(project + ".old"); err != nil {
			t.Error(err)
		}
	})
	if err := os.Mkdir(project, 0700); err != nil {
		t.Fatal(err)
	}
	if err := b.Check(); err == nil {
		t.Fatal("replaced project identity accepted")
	}
}
