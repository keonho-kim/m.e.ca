// Purpose: Verify release refresh replaces edits, skips identical files and protects linked targets.
// Pattern: Private filesystem behavior tests.
// Usage: go test ./internal/config/prompts -run TestRefresh.
// Related: internal/config/prompts/refresh.go, internal/config/prompts/prompts.go.
package prompts_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/prompt"
	"github.com/keonho-kim/meca/internal/config/bootstrap"
	"github.com/keonho-kim/meca/internal/config/prompts"
	"github.com/keonho-kim/meca/internal/config/statefs"
)

func TestRefreshReplacesEditsAndSkipsIdenticalFiles(t *testing.T) {
	home := t.TempDir()
	if _, err := bootstrap.Initialize(home); err != nil {
		t.Fatal(err)
	}
	root, err := statefs.Open(filepath.Join(home, ".meca"))
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	path := filepath.Join(root.Name(), "prompts/system.md")
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := prompts.Refresh(root); err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(path)
	if err != nil || !os.SameFile(before, after) {
		t.Fatalf("identical default was rewritten: %v", err)
	}
	if err := os.WriteFile(path, []byte("custom prompt"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := prompts.Refresh(root); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(path)
	if err != nil || string(body) != prompt.Defaults().System {
		t.Fatalf("edited prompt was not replaced: %v", err)
	}
}

func TestRefreshRefusesLinkedPrompt(t *testing.T) {
	home := t.TempDir()
	if _, err := bootstrap.Initialize(home); err != nil {
		t.Fatal(err)
	}
	root, err := statefs.Open(filepath.Join(home, ".meca"))
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	outside := filepath.Join(t.TempDir(), "user-file")
	if err := os.WriteFile(outside, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root.Name(), "prompts/system.md")
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, path); err != nil {
		t.Fatal(err)
	}
	if err := prompts.Refresh(root); err == nil {
		t.Fatal("refresh accepted a linked prompt")
	}
	body, err := os.ReadFile(outside)
	if err != nil || string(body) != "keep" {
		t.Fatal("refresh modified the linked target")
	}
}
