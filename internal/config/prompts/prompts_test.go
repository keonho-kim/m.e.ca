// Purpose: Verify bootstrap publishes current private Markdown defaults and preserves user edits.
// Pattern: Filesystem behavior tests.
// Usage: go test ./internal/config/prompts -run TestPromptBootstrap.
// Related: internal/config/prompts/prompts.go, internal/agent/prompt/defaults.go.
package prompts_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/prompt"
	"github.com/keonho-kim/meca/internal/config/prompts"
	"github.com/keonho-kim/meca/internal/config/statefs"
)

func TestPromptBootstrap(t *testing.T) {
	root := promptRoot(t)
	directory := root.Name()
	entries, err := os.ReadDir(filepath.Join(directory, "prompts"))
	if err != nil || len(entries) != 2 {
		t.Fatalf("expected two prompt profiles: %v %v", entries, err)
	}
	defaults := prompt.Defaults()
	files := defaults.Files()
	for _, name := range []string{"system.md", "compaction.md"} {
		path := filepath.Join(directory, "prompts", "single", name)
		body, err := os.ReadFile(path)
		if err != nil || len(body) == 0 {
			t.Fatalf("%s: %v", name, err)
		}
		if string(body) != *files[name] {
			t.Fatalf("%s did not publish the current embedded default", name)
		}
		info, err := os.Stat(path)
		if err != nil || info.Mode().Perm() != 0600 {
			t.Fatalf("prompt permissions: %v %v", info, err)
		}
	}
	path := filepath.Join(directory, "prompts/single/system.md")
	if err := os.WriteFile(path, []byte("# My prompt\nKeep responses concise.\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := prompts.Bootstrap(root); err != nil {
		t.Fatal(err)
	}
	templates, err := prompts.Load(directory)
	if err != nil || !strings.Contains(templates.System, "My prompt") {
		t.Fatalf("custom prompt lost: %+v %v", templates, err)
	}
}

func promptRoot(t *testing.T) *os.Root {
	t.Helper()
	root, err := statefs.Open(filepath.Join(t.TempDir(), statefs.DirectoryName))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := root.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := prompts.Bootstrap(root); err != nil {
		t.Fatal(err)
	}
	return root
}
