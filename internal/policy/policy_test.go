// Purpose: Verify approval and path-containment policy at the tool boundary.
// Pattern: Behavioral tests.
// Usage: go test ./internal/policy.
// Related: internal/policy/policy.go, internal/policy/path.go.

package policy

import (
	"path/filepath"
	"testing"
)

func TestDecidePreservesScopeAndPermission(t *testing.T) {
	for _, test := range []struct {
		settings Settings
		effect   Effect
		approval bool
		denied   bool
	}{
		{Settings{Approve, Directory}, Read, false, false},
		{Settings{Approve, Directory}, Change, true, false},
		{Settings{AllowAll, Directory}, Change, false, false},
		{Settings{AllowAll, Directory}, Command, false, false},
		{Settings{Approve, System}, Command, true, false},
		{Settings{AllowAll, System}, Command, false, false},
		{Settings{"invalid", Directory}, Read, false, true},
		{Settings{Approve, Directory}, Effect(99), false, true},
	} {
		approval, err := Decide(test.settings, test.effect)
		if (err != nil) != test.denied || approval != test.approval {
			t.Errorf("Decide(%+v, %v) = %v, %v", test.settings, test.effect, approval, err)
		}
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
