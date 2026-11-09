// Purpose: Verify installation seeds the authoring skill and preserves its user edits on reinstall.
// Pattern: Installation lifecycle contract test.
// Usage: go test ./internal/commands/install -run TestInstallPreservesDefaultSkill.
// Related: internal/commands/install/install.go, internal/config/bootstrap/skills.go.
package install

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestInstallPreservesDefaultSkill(t *testing.T) {
	options := options(t, "/bin/zsh")
	ctx := context.Background()
	if _, err := Install(ctx, options); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(options.Home, ".meca/skills/create-skill/SKILL.md")
	if body, err := os.ReadFile(path); err != nil || len(body) == 0 {
		t.Fatalf("default skill missing: %v", err)
	}
	custom := []byte("---\nname: create-skill\ndescription: Customized skill authoring\n---\nKeep the user's instructions.\n")
	if err := os.WriteFile(path, custom, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(ctx, options); err != nil {
		t.Fatal(err)
	}
	if body, err := os.ReadFile(path); err != nil || string(body) != string(custom) {
		t.Fatalf("installation replaced the user's skill: %q %v", body, err)
	}
}
