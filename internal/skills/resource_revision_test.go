// Purpose: Detect mixed-version reference pages across separate reads.
// Pattern: Isolated mutable-file pagination probe.
// Usage: go test ./internal/skills.
// Related: internal/skills/resources.go.
package skills

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReferencePagesVersion(t *testing.T) {
	r, project, _ := skillFixture(t)
	dir := writeSkill(t, filepath.Join(project, ".agents/skills"), "pages", "Reference pages", "Instructions")
	path := filepath.Join(dir, "reference.md")
	old := strings.Repeat("a", MaxResourceBytes) + strings.Repeat("b", MaxResourceBytes)
	if err := os.WriteFile(path, []byte(old), 0600); err != nil {
		t.Fatal(err)
	}
	first, err := r.ReadFile(t.Context(), "project:pages", "reference.md", 0, MaxResourceBytes, "")
	if err != nil {
		t.Fatal(err)
	}
	if first.Revision == "" || first.NextOffset == 0 {
		t.Fatal("missing continuation", first)
	}
	if _, err := r.ReadFile(t.Context(), "project:pages", "reference.md", first.NextOffset, MaxResourceBytes, ""); !errors.Is(err, ErrInvalid) {
		t.Fatal("unversioned continuation accepted", err)
	}
	stable, err := r.ReadFile(t.Context(), "project:pages", "reference.md", first.NextOffset, MaxResourceBytes, first.Revision)
	if err != nil || first.Text+stable.Text != old {
		t.Fatal("stable continuation failed", err)
	}
	changed := strings.Repeat("c", MaxResourceBytes) + strings.Repeat("d", MaxResourceBytes)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(changed), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	second, err := r.ReadFile(t.Context(), "project:pages", "reference.md", first.NextOffset, MaxResourceBytes, first.Revision)
	if !errors.Is(err, ErrInvalid) || second.Text != "" {
		t.Fatal("same-size edit produced accepted hybrid pages", err)
	}
}
