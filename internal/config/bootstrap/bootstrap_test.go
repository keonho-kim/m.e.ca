// Purpose: Verify repeatable bootstrap preserves user settings and refuses symlink state.
// Pattern: Filesystem behavior tests.
// Usage: go test ./internal/config/bootstrap -run TestBootstrap.
// Related: internal/config/bootstrap/bootstrap.go.
package bootstrap

import (
	"github.com/keonho-kim/v-code/internal/config"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBootstrap(t *testing.T) {
	home := t.TempDir()
	directory := filepath.Join(home, ".v")
	if _, err := Initialize(home); err != nil {
		t.Fatal(err)
	}
	content := []byte("[ui]\ncolumn_width=32\n")
	if err := os.WriteFile(filepath.Join(directory, config.Filename), content, 0600); err != nil {
		t.Fatal(err)
	}
	c, err := Initialize(home)
	if err != nil || c.UI.ColumnWidth != 32 {
		t.Fatalf("overwrote configuration: %+v %v", c, err)
	}
	other := t.TempDir()
	if err := os.Symlink(directory, filepath.Join(other, ".v")); err != nil {
		t.Fatal(err)
	}
	if _, err := Initialize(other); err == nil {
		t.Fatal("accepted symlink state")
	}
}

func TestConcurrentBootstrapPublishesCompleteConfiguration(t *testing.T) {
	home := t.TempDir()
	results := make(chan error, 8)
	for n := 0; n < 8; n++ {
		go func() { _, err := Initialize(home); results <- err }()
	}
	for n := 0; n < 8; n++ {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
}

func TestBootstrapCreatesSessionDirectoryWithoutRegistry(t *testing.T) {
	home := t.TempDir()
	if _, err := Initialize(home); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(home, ".v")
	info, err := os.Lstat(filepath.Join(root, "sessions"))
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0700 {
		t.Fatalf("session directory: %v %v", info, err)
	}
	if _, err := os.Stat(filepath.Join(root, "v.db")); !os.IsNotExist(err) {
		t.Fatalf("created central registry: %v", err)
	}
	if err := os.Remove(filepath.Join(root, "sessions")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(t.TempDir(), filepath.Join(root, "sessions")); err != nil {
		t.Fatal(err)
	}
	if _, err := Initialize(home); err == nil {
		t.Fatal("accepted symlink session directory")
	}
}

func TestBootstrapRecordsRoleDefaults(t *testing.T) {
	home := t.TempDir()
	if _, err := Initialize(home); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(home, ".v", config.Filename))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "language = 'auto'") {
		t.Fatal("default interface language missing from bootstrap configuration")
	}
	if strings.Contains(string(content), "model_profiles") || strings.Contains(strings.Split(string(content), "[agent]")[0], "thinking =") {
		t.Fatal("bootstrap did not preserve server reasoning defaults")
	}

	for _, role := range []string{"agent", "compaction", "diagnostics"} {
		if !strings.Contains(string(content), "["+role+"]") {
			t.Fatalf("missing role defaults: %s", role)
		}
	}
}
