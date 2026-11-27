// Purpose: Verify the installed development skill and its engineering references are readable.
// Pattern: Installation lifecycle contract test.
// Usage: go test ./internal/commands/install -run TestInstallPublishesDevelopmentSkill.
// Related: internal/commands/install/install.go, internal/config/bootstrap/skills.go.
package install

import (
	"testing"

	"github.com/keonho-kim/meca/internal/skills"
	"github.com/keonho-kim/meca/internal/skills/instruction"
)

func TestInstallPublishesDevelopmentSkill(t *testing.T) {
	options := options(t, "/bin/zsh")
	ctx := t.Context()
	if _, err := Install(ctx, options); err != nil {
		t.Fatal(err)
	}
	reader := skills.New(t.TempDir(), options.Home, skills.Options{Redact: func(s string) string { return s }, Check: func(string) error { return nil }})
	selector := instruction.Selector{ID: "meca:development"}
	doc, err := reader.Load(ctx, selector, false)
	if err != nil || doc.Name != "development" || doc.Body == "" {
		t.Fatalf("cannot load the installed development skill: %v", err)
	}
	for _, path := range []string{"references/team-workflow.md", "references/algorithms/data-structures.md"} {
		resource, err := reader.ReadFile(ctx, doc.ID, path, 0, skills.MaxResourceBytes, "")
		if err != nil || resource.Text == "" || resource.NextOffset != 0 {
			t.Fatalf("cannot read the full installed reference %s: %v", path, err)
		}
	}
}
