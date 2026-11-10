// Purpose: Verify explicit folder creation uses the existing private-path boundary and never overwrites entries.
// Pattern: Filesystem boundary integration tests.
// Usage: go test ./internal/tools/local -run TestExplicitFolderCreation.
// Related: internal/fileview/directory_create.go, internal/tools/local/filesystem/paths.go.
package local

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/keonho-kim/meca/internal/policy"
)

func TestExplicitFolderCreation(t *testing.T) {
	home := t.TempDir()
	project, state := filepath.Join(home, "project"), filepath.Join(home, ".meca")
	if err := os.Mkdir(project, 0700); err != nil {
		t.Fatal(err)
	}
	executor, err := NewExecutor(project, home, state, func(s string) string { return s }, Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer executor.Close()
	settings := policy.Settings{Scope: policy.System}
	path, err := executor.Files.CreateDirectory(t.Context(), settings, home, "새 프로젝트")
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0700 {
		t.Fatal("new directory was not created privately", err)
	}
	if _, err := executor.Files.CreateDirectory(t.Context(), settings, home, "새 프로젝트"); err == nil {
		t.Fatal("existing folder was overwritten")
	}
	for _, name := range []string{"", " ", ".", "..", "nested/name", "nested\\name", ".meca", ".ssh"} {
		if _, err := executor.Files.CreateDirectory(t.Context(), settings, home, name); err == nil {
			t.Errorf("unsafe folder name accepted: %q", name)
		}
	}
	private := filepath.Join(home, ".ssh")
	if err := os.Mkdir(private, 0700); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(home, "private-alias")
	if err := os.Symlink(private, alias); err != nil {
		t.Fatal(err)
	}
	if _, err := executor.Files.CreateDirectory(t.Context(), settings, alias, "blocked"); err == nil {
		t.Fatal("a symlink bypassed the protected parent boundary")
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := executor.Files.CreateDirectory(ctx, settings, home, "cancelled"); !errors.Is(err, context.Canceled) {
		t.Fatal("cancelled creation was not rejected", err)
	}
	if _, err := os.Stat(filepath.Join(home, "cancelled")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("cancelled creation changed the filesystem", err)
	}
}
