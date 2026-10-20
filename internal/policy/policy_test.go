// Purpose: Verify file Scope validation and project-relative path containment.
// Pattern: Behavioral tests.
// Usage: go test ./internal/policy.
// Related: internal/policy/policy.go, internal/policy/path.go.

package policy

import (
	"path/filepath"
	"testing"
)

func TestScopeSettings(t *testing.T) {
	for _, scope := range []Scope{Directory, System} {
		if err := (Settings{Scope: scope}).Check(); err != nil {
			t.Fatal(err)
		}
	}
	if err := (Settings{Scope: "invalid"}).Check(); err == nil {
		t.Fatal("invalid Scope accepted")
	}
}

func TestResolvePathsRelativeToProject(t *testing.T) {
	project := filepath.Join(t.TempDir(), "project")
	home := t.TempDir()
	abs, relative, err := Resolve(project, home, Directory, "src/auth.go")
	if err != nil || abs != filepath.Join(project, "src/auth.go") || relative != filepath.Join("src", "auth.go") {
		t.Fatalf("unexpected path %q, %q, %v", abs, relative, err)
	}
	for _, path := range []string{"../outside", project + "-other/secret", "~/private", "bad\x00name"} {
		if _, _, err := Resolve(project, home, Directory, path); err == nil {
			t.Errorf("accepted scope escape %q", path)
		}
	}
	abs, _, err = Resolve(project, home, System, "~/private")
	if err != nil || abs != filepath.Join(home, "private") {
		t.Fatalf("home expansion: %q, %v", abs, err)
	}
}
