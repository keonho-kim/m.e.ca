// Purpose: Reproduce whether a discovered symlink skill can be selected by its completion name.
// Pattern: filesystem contract regression test.
// Usage: go test ./internal/skills.
// Related: internal/skills/suggestions.go, internal/skills/files.go, internal/skills/reader.go.
package skills

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/keonho-kim/meca/internal/skills/instruction"
)

func TestAliasSkillSuggestionLoadsByName(t *testing.T) {
	r, _, home := skillFixture(t)
	target := writeSkill(t, t.TempDir(), "canonical", "Canonical skill", "Instructions")
	root := filepath.Join(home, ".agents", "skills")
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(root, "alias")); err != nil {
		t.Fatal(err)
	}
	hints, err := r.Suggestions(context.Background(), "can")
	if err != nil || len(hints) != 1 {
		t.Fatalf("hints=%+v err=%v", hints, err)
	}
	t.Logf("suggested name=%q id=%q", hints[0].Name, hints[0].ID)
	if _, err := r.Load(context.Background(), instruction.Selector{ID: hints[0].Name}, true); err != nil {
		t.Fatalf("loading the suggested name failed: %v", err)
	}
}

func TestSkillNameSeesNewHigherPrioritySource(t *testing.T) {
	r, project, home := skillFixture(t)
	r.now = func() time.Time { return time.Unix(1000, 0) }
	writeSkill(t, filepath.Join(home, ".meca/skills"), "canonical", "Global", "Global instructions")
	if _, err := r.Load(t.Context(), instruction.Selector{ID: "canonical"}, true); err != nil {
		t.Fatal(err)
	}
	writeSkill(t, filepath.Join(project, ".agents/skills"), "canonical", "Project", "Project instructions")
	doc, err := r.Load(t.Context(), instruction.Selector{ID: "canonical"}, true)
	if err != nil || doc.Source != "project" {
		t.Fatalf("new project skill did not take priority: %+v %v", doc.Metadata, err)
	}
}
