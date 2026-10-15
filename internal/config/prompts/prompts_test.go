// Purpose: Verify bootstrap creates separate private Markdown prompts without overwriting user edits.
// Pattern: Filesystem behavior tests.
// Usage: go test ./internal/config/prompts -run TestPromptBootstrap.
// Related: internal/config/prompts/prompts.go, internal/agent/prompt/defaults.go.
package prompts_test

import (
	"github.com/keonho-kim/v-code/internal/config/bootstrap"
	"github.com/keonho-kim/v-code/internal/config/prompts"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPromptBootstrap(t *testing.T) {
	home := t.TempDir()
	if _, err := bootstrap.Initialize(home); err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(home, ".v")
	for _, name := range []string{"system.md", "compaction.md", "conclave.md"} {
		path := filepath.Join(directory, "prompts", name)
		body, err := os.ReadFile(path)
		if err != nil || len(body) == 0 {
			t.Fatalf("%s: %v", name, err)
		}
		info, err := os.Stat(path)
		if err != nil || info.Mode().Perm() != 0600 {
			t.Fatalf("prompt permissions: %v %v", info, err)
		}
	}
	path := filepath.Join(directory, "prompts/system.md")
	if err := os.WriteFile(path, []byte("# My prompt\nKeep responses concise.\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := bootstrap.Initialize(home); err != nil {
		t.Fatal(err)
	}
	templates, err := prompts.Load(directory)
	if err != nil || !strings.Contains(templates.System, "My prompt") {
		t.Fatalf("custom prompt lost: %+v %v", templates, err)
	}
}
