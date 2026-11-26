// Purpose: Verify repeatable bootstrap preserves user settings and refuses symlink state.
// Pattern: Filesystem behavior tests.
// Usage: go test ./internal/config/bootstrap -run TestBootstrap.
// Related: internal/config/bootstrap/bootstrap.go.
package bootstrap

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/keonho-kim/meca/internal/config"
)

func TestBootstrap(t *testing.T) {
	home := t.TempDir()
	directory := filepath.Join(home, ".meca")
	if _, err := Initialize(t.Context(), home); err != nil {
		t.Fatal(err)
	}
	content := []byte("[ui]\ncolumn_width=32\n")
	if err := os.WriteFile(filepath.Join(directory, config.Filename), content, 0600); err != nil {
		t.Fatal(err)
	}
	c, err := Initialize(t.Context(), home)
	if err != nil || c.UI.ColumnWidth != 32 {
		t.Fatalf("overwrote configuration: %+v %v", c, err)
	}
	other := t.TempDir()
	if err := os.Symlink(directory, filepath.Join(other, ".meca")); err != nil {
		t.Fatal(err)
	}
	if _, err := Initialize(t.Context(), other); err == nil {
		t.Fatal("accepted symlink state")
	}
}

func TestConcurrentBootstrapPublishesCompleteConfiguration(t *testing.T) {
	home := t.TempDir()
	results := make(chan error, 8)
	for n := 0; n < 8; n++ {
		go func() { _, err := Initialize(t.Context(), home); results <- err }()
	}
	for n := 0; n < 8; n++ {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
	readDefaultSkill(t, home)
	readDevelopmentSkill(t, home)
	entries, err := os.ReadDir(filepath.Join(home, ".meca/skills/create-skill"))
	if err != nil || len(entries) != 1 || entries[0].Name() != "SKILL.md" {
		t.Fatalf("incomplete or temporary skill files: %v %v", entries, err)
	}
}

func TestBootstrapCreatesPrivateSessionDirectory(t *testing.T) {
	home := t.TempDir()
	if _, err := Initialize(t.Context(), home); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(home, ".meca")
	info, err := os.Lstat(filepath.Join(root, "sessions"))
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0700 {
		t.Fatalf("session directory: %v %v", info, err)
	}
	if err := os.Remove(filepath.Join(root, "sessions")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(t.TempDir(), filepath.Join(root, "sessions")); err != nil {
		t.Fatal(err)
	}
	if _, err := Initialize(t.Context(), home); err == nil {
		t.Fatal("accepted symlink session directory")
	}
}
