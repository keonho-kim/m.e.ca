// Purpose: Verify only bounded project-root AGENTS.md instructions can enter model context.
// Pattern: Filesystem trust-boundary tests.
// Usage: go test ./internal/tools/local -run TestProjectInstructions.
// Related: internal/tools/local/instructions.go, internal/agent/prompt/prompt.go.
package local

import (
	"context"
	"github.com/keonho-kim/v-code/internal/agent/prompt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProjectInstructions(t *testing.T) {
	e, root := testExecutor(t)
	if text, err := e.ProjectInstructions(context.Background()); err != nil || text != "" {
		t.Fatalf("missing file: %q %v", text, err)
	}
	path := filepath.Join(root, "AGENTS.md")
	if err := os.WriteFile(path, []byte("Use tests. provider-secret"), 0600); err != nil {
		t.Fatal(err)
	}
	if text, err := e.ProjectInstructions(context.Background()); err != nil || text != "Use tests. [redacted]" {
		t.Fatalf("project rules: %q %v", text, err)
	}
	if err := os.WriteFile(path, []byte(strings.Repeat("a", prompt.MaxInstructionsBytes+1)), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := e.ProjectInstructions(context.Background()); err == nil {
		t.Fatal("oversized instructions accepted")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "AGENTS.md")
	if err := os.WriteFile(outside, []byte("outside"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, path); err != nil {
		t.Fatal(err)
	}
	if _, err := e.ProjectInstructions(context.Background()); err == nil {
		t.Fatal("outside project instructions loaded")
	}
}
