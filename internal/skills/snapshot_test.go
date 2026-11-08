// Purpose: Preserve one catalog page sequence without rescans while selected reads inspect current files.
// Pattern: Filesystem discovery and snapshot contract tests.
// Usage: go test -race ./internal/skills -run TestSnapshot.
// Related: internal/skills/snapshot.go, internal/skills/catalog.go, internal/skills/reader.go.
package skills

import (
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/keonho-kim/meca/internal/skills/instruction"
)

func TestSnapshotPagesRemainIndependentWhenCatalogRefreshes(t *testing.T) {
	project, home := t.TempDir(), t.TempDir()
	var checks atomic.Int64
	r := New(project, home, Options{Redact: func(v string) string { return v }, Check: func(string) error { checks.Add(1); return nil }})
	root := filepath.Join(project, ".agents/skills")
	for _, name := range []string{"aaa", "bbb", "ccc"} {
		writeSkill(t, root, name, "Description", "Body")
	}
	first, err := r.List(t.Context(), "", 1)
	if err != nil {
		t.Fatal(err)
	}
	count := checks.Load()
	writeSkill(t, root, "aab", "New entry", "New body")
	second, err := r.List(t.Context(), first.NextCursor, 1)
	if err != nil || second.Items[0].Name != "bbb" || checks.Load() != count {
		t.Fatal(second, err, checks.Load(), count)
	}
	if _, err := r.List(t.Context(), "", 1); err != nil {
		t.Fatal(err)
	}
	if page, err := r.List(t.Context(), first.NextCursor, 1); err != nil || page.Items[0].Name != "bbb" {
		t.Fatal("another traversal invalidated an existing cursor", page, err)
	}
}

func TestSnapshotDoesNotAuthorizeStaleInstructions(t *testing.T) {
	r, project, home := skillFixture(t)
	writeSkill(t, filepath.Join(home, ".meca/skills"), "review", "Global", "Old")
	if _, err := r.Suggestions(t.Context(), "rev"); err != nil {
		t.Fatal(err)
	}
	local := writeSkill(t, filepath.Join(project, ".agents/skills"), "review", "Project", "Fresh")
	value, err := r.Load(t.Context(), instruction.Selector{ID: "review"}, false)
	if err != nil || value.Source != "project" || !strings.Contains(value.Body, "Fresh") {
		t.Fatal(value, err)
	}
	if err := os.WriteFile(filepath.Join(local, "SKILL.md"), []byte("---\nname: review\ndescription: Updated\ndisable-model-invocation: true\n---\nChanged"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Load(t.Context(), instruction.Selector{ID: "review"}, false); err != ErrManual {
		t.Fatal("cached policy replaced current policy", err)
	}
}
