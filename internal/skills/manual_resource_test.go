// Purpose: Check the manual-only read boundary during a review.
// Pattern: Behavior regression test.
// Usage: go test ./internal/skills.
// Related: internal/skills/assets.go.
package skills

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/skills/instruction"
)

func TestManualOnlyResourceRead(t *testing.T) {
	r, project, _ := skillFixture(t)
	dir := writeSkill(t, filepath.Join(project, ".agents/skills"), "manual", "Manual only", "manual-body-canary")
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: manual\ndescription: Manual only\ndisable-model-invocation: true\n---\nmanual-body-canary"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Load(t.Context(), instruction.Selector{ID: "project:manual"}, false); err != ErrManual {
		t.Fatalf("unexpected load: %v", err)
	}
	got, err := r.ReadFile(t.Context(), "project:manual", "SKILL.md", 0, MaxResourceBytes, "")
	if !errors.Is(err, ErrManual) || got.Text != "" {
		t.Fatal("manual_only blocks load_skill but the same unapproved body is readable as a resource")
	}
	ctx := instruction.WithExplicitReads(t.Context(), []string{"project:manual"})
	got, err = r.ReadFile(ctx, "manual", "SKILL.md", 0, MaxResourceBytes, "")
	if err != nil || !strings.Contains(got.Text, "manual-body-canary") {
		t.Fatal("explicit read rejected", got, err)
	}
	if _, err := r.ReadFile(t.Context(), "project:manual", "SKILL.md", 0, MaxResourceBytes, ""); !errors.Is(err, ErrManual) {
		t.Fatal("read grant escaped its context", err)
	}
	preview := instruction.WithExplicitReads(t.Context(), []string{"manual"})
	if _, err := r.ReadFile(preview, "manual", "SKILL.md", 0, MaxResourceBytes, ""); err != nil {
		t.Fatal("explicit unqualified preview rejected", err)
	}
}
