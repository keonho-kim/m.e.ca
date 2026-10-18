// Purpose: Verify write roots, protected paths and pinned directory identities.
// Pattern: Filesystem policy contract tests.
// Usage: go test ./internal/tools/local/filesystem.
// Related: internal/tools/local/filesystem/boundary.go.
package filesystem

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestBoundary(t *testing.T) {
	base := t.TempDir()
	project, home, cache := filepath.Join(base, "project"), filepath.Join(base, "home"), filepath.Join(base, "cache")
	for _, p := range []string{project, home, cache} {
		if err := os.Mkdir(p, 0700); err != nil {
			t.Fatal(err)
		}
	}
	b, err := New(project, home, filepath.Join(home, ".v"), []string{cache})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{filepath.Join(project, "new"), filepath.Join(cache, "new")} {
		if _, _, err := b.WriteTarget(p); err != nil {
			t.Fatal(err)
		}
	}
	for _, p := range []string{filepath.Join(home, "new"), filepath.Join(home, ".v/config.toml"), filepath.Join(home, ".ssh/key")} {
		if _, _, err := b.WriteTarget(p); err == nil {
			t.Fatalf("accepted write: %s", p)
		}
	}
	if _, err := b.ReadTarget(filepath.Join(home, ".ssh/key")); err == nil {
		t.Fatal("secret exposed")
	}
	if err := os.Symlink(home, filepath.Join(project, "linked")); err != nil {
		t.Fatal(err)
	}
	if _, _, err := b.WriteTarget(filepath.Join(project, "linked/new")); err == nil {
		t.Fatal("symlink escape")
	}
	for _, p := range []string{"/", home, "/usr", filepath.Join(home, ".v")} {
		if _, err := New(project, home, filepath.Join(home, ".v"), []string{p}); err == nil {
			t.Fatalf("broad exception accepted: %s", p)
		}
	}
	if err := os.Rename(cache, cache+".old"); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(cache, 0700); err != nil {
		t.Fatal(err)
	}
	if err := b.Check(); err == nil {
		t.Fatal("replaced root accepted")
	}
}

func TestHardlinkBlocksShellPreparation(t *testing.T) {
	base := t.TempDir()
	project := filepath.Join(base, "project")
	home := filepath.Join(base, "home")
	for _, p := range []string{project, home} {
		if err := os.Mkdir(p, 0700); err != nil {
			t.Fatal(err)
		}
	}
	original := filepath.Join(base, "outside")
	if err := os.WriteFile(original, []byte("safe"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(original, filepath.Join(project, "alias")); err != nil {
		t.Fatal(err)
	}
	b, err := New(project, home, filepath.Join(home, ".v"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := b.CheckTrees(context.Background()); err == nil {
		t.Fatal("shared inode accepted")
	}
}
