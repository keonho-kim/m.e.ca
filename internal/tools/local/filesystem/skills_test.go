// Purpose: Verify the skill exception cannot expose other protected state through aliases.
// Pattern: Filesystem security contract tests.
// Usage: go test ./internal/tools/local/filesystem -run TestSkill.
// Related: internal/tools/local/filesystem/skills.go.
package filesystem

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSkillReadExceptionIsNarrow(t *testing.T) {
	project, home := t.TempDir(), t.TempDir()
	state := filepath.Join(home, ".meca")
	if err := os.MkdirAll(filepath.Join(state, "skills"), 0700); err != nil {
		t.Fatal(err)
	}
	b, err := New(project, home, state)
	if err != nil {
		t.Fatal(err)
	}
	if err := b.SkillTarget(filepath.Join(state, "skills")); err != nil {
		t.Fatal(err)
	}
	if _, err := b.ReadTarget(filepath.Join(state, "skills")); err == nil {
		t.Fatal("generic file access was weakened")
	}
	if err := os.MkdirAll(filepath.Join(state, "credentials"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(state, "credentials"), filepath.Join(state, "skills", "escape")); err != nil {
		t.Fatal(err)
	}
	if err := b.SkillTarget(filepath.Join(state, "skills", "escape")); err == nil {
		t.Fatal("skill alias exposed private credentials")
	}
	if err := os.MkdirAll(filepath.Join(home, ".ssh"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(home, ".ssh"), filepath.Join(state, "skills", "keys")); err != nil {
		t.Fatal(err)
	}
	if err := b.SkillTarget(filepath.Join(state, "skills", "keys")); err == nil {
		t.Fatal("skill alias exposed protected host files")
	}
}
