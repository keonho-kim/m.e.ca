// Purpose: Verify repeatable bootstrap preserves user settings and refuses symlink state.
// Pattern: Filesystem behavior tests.
// Usage: go test ./internal/config/bootstrap -run TestBootstrap.
// Related: internal/config/bootstrap/bootstrap.go.
package bootstrap

import (
	"github.com/keonho-kim/v-code/internal/config"
	"os"
	"path/filepath"
	"testing"
)

func TestBootstrap(t *testing.T) {
	home := t.TempDir()
	directory := filepath.Join(home, ".v")
	if _, err := Initialize(home); err != nil {
		t.Fatal(err)
	}
	content := []byte("[session]\nrestore_last=false\n")
	if err := os.WriteFile(filepath.Join(directory, config.Filename), content, 0600); err != nil {
		t.Fatal(err)
	}
	c, err := Initialize(home)
	if err != nil || c.Session.RestoreLast {
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
