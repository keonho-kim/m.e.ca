// Purpose: Verify reinstall refreshes bundled skill content while preserving independent user files.
// Pattern: Installation filesystem and skill-reader integration tests.
// Usage: go test ./internal/commands/install -run 'TestReinstall.*BuiltinSkill'.
// Related: internal/commands/install/install.go, internal/config/bootstrap/skills.go, internal/skills/reader.go.
package install

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/keonho-kim/meca/internal/skills"
	"github.com/keonho-kim/meca/internal/skills/instruction"
)

func TestReinstallRefreshesBuiltinSkills(t *testing.T) {
	o := options(t, "/bin/sh")
	if _, err := Install(t.Context(), o); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(o.Home, ".meca", "skills")
	original := map[string][]byte{}
	for _, name := range []string{
		"development/SKILL.md", "create-skill/SKILL.md",
		"development/references/team-workflow.md", "development/scripts/search.sh",
	} {
		path := filepath.Join(root, name)
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		original[name] = body
		if err := os.WriteFile(path, []byte("previous installed instructions\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"personal/SKILL.md", "development/local-notes.md"} {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("user-owned guidance\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := Install(t.Context(), o); err != nil {
		t.Fatal(err)
	}
	for name, expected := range original {
		path := filepath.Join(root, name)
		body, err := os.ReadFile(path)
		if err != nil || string(body) != string(expected) {
			t.Fatalf("reinstall did not publish the current bundled %s: %v", name, err)
		}
		info, err := os.Stat(path)
		if err != nil || info.Mode().Perm() != 0600 {
			t.Fatalf("refreshed skill permissions: %v %v", info, err)
		}
	}
	reader := skills.New(t.TempDir(), o.Home, skills.Options{
		Redact: func(s string) string { return s }, Check: func(string) error { return nil },
	})
	doc, err := reader.Load(t.Context(), instruction.Selector{ID: "meca:development"}, false)
	if err != nil || doc.Source != "meca" || doc.Body == "" {
		t.Fatalf("refreshed development skill is not usable: %+v %v", doc, err)
	}
	page, err := reader.ReadFile(t.Context(), doc.ID, "references/team-workflow.md", 0, skills.MaxResourceBytes, "")
	if err != nil || page.Text != string(original["development/references/team-workflow.md"]) {
		t.Fatalf("reader did not observe the refreshed reference: %+v %v", page, err)
	}
	for _, name := range []string{"personal/SKILL.md", "development/local-notes.md"} {
		body, err := os.ReadFile(filepath.Join(root, name))
		if err != nil || string(body) != "user-owned guidance\n" {
			t.Fatalf("changed independent user resource %s: %v", name, err)
		}
	}
}

func TestReinstallRefusesLinkedBuiltinSkill(t *testing.T) {
	o := options(t, "/bin/sh")
	if _, err := Install(t.Context(), o); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "user-file")
	if err := os.WriteFile(outside, []byte("keep this content"), 0600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(o.Home, ".meca/skills/development/SKILL.md")
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, path); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(t.Context(), o); err == nil {
		t.Fatal("reinstall accepted a linked bundled skill")
	}
	body, err := os.ReadFile(outside)
	if err != nil || string(body) != "keep this content" {
		t.Fatalf("reinstall changed the linked target: %q %v", body, err)
	}
	target, err := os.Readlink(path)
	if err != nil || target != outside {
		t.Fatalf("reinstall replaced the linked skill: %q %v", target, err)
	}
}
