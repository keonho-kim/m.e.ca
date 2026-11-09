// Purpose: Verify default skill publication, privacy and preservation during initialization.
// Pattern: Filesystem contract tests.
// Usage: go test ./internal/config/bootstrap -run 'TestBootstrap.*Skill|TestConcurrentBootstrap'.
// Related: internal/config/bootstrap/skills.go, internal/config/bootstrap/bootstrap.go, internal/skills/reader.go.
package bootstrap

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/keonho-kim/meca/internal/skills"
	"github.com/keonho-kim/meca/internal/skills/instruction"
)

func readDefaultSkill(t *testing.T, home string) instruction.Document {
	t.Helper()
	reader := skills.New(t.TempDir(), home, skills.Options{Redact: func(s string) string { return s }, Check: func(string) error { return nil }})
	document, err := reader.Load(context.Background(), instruction.Selector{ID: "meca:create-skill"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if document.Name != "create-skill" || document.Source != "meca" || document.Description == "" || document.Body == "" || document.Revision == "" {
		t.Fatalf("invalid default skill: %+v", document)
	}
	return document
}

func TestBootstrapCreatesPrivateDefaultSkill(t *testing.T) {
	home := t.TempDir()
	if _, err := Initialize(home); err != nil {
		t.Fatal(err)
	}
	for name, mode := range map[string]os.FileMode{
		"skills": 0700, "skills/create-skill": 0700, "skills/create-skill/SKILL.md": 0600,
	} {
		info, err := os.Lstat(filepath.Join(home, ".meca", name))
		if err != nil || info.Mode().Perm() != mode || info.IsDir() != (mode == 0700) {
			t.Fatalf("%s: %v %v", name, info, err)
		}
	}
	readDefaultSkill(t, home)
	if _, err := os.Stat(filepath.Join(home, ".agents")); !os.IsNotExist(err) {
		t.Fatalf("bootstrap changed the shared skill directory: %v", err)
	}
}

func TestBootstrapPreservesAndRestoresDefaultSkill(t *testing.T) {
	home := t.TempDir()
	if _, err := Initialize(home); err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(home, ".meca/skills/create-skill")
	manifest := filepath.Join(directory, "SKILL.md")
	custom := "---\nname: create-skill\ndescription: Custom authoring instructions\n---\nKeep this user edit.\n"
	if err := os.WriteFile(manifest, []byte(custom), 0600); err != nil {
		t.Fatal(err)
	}
	note := filepath.Join(directory, "note.txt")
	if err := os.WriteFile(note, []byte("user resource"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Initialize(home); err != nil {
		t.Fatal(err)
	}
	if body, err := os.ReadFile(manifest); err != nil || string(body) != custom {
		t.Fatalf("custom skill overwritten: %q %v", body, err)
	}
	if err := os.Remove(manifest); err != nil {
		t.Fatal(err)
	}
	if _, err := Initialize(home); err != nil {
		t.Fatal(err)
	}
	readDefaultSkill(t, home)
	if body, err := os.ReadFile(note); err != nil || string(body) != "user resource" {
		t.Fatalf("user resource lost: %q %v", body, err)
	}
}

func TestBootstrapRejectsUnsafeSkillDirectories(t *testing.T) {
	for _, name := range []string{
		"skills",
		"skills/create-skill",
		"skills/backend-design",
		"skills/backend-design/references",
		"skills/backend-design/references/storage",
		"skills/ui-design",
		"skills/ui-design/references",
		"skills/ui-design/references/themes",
		"skills/ui-design/references/layouts",
		"skills/ui-design/references/components",
		"skills/ui-design/references/loading",
		"skills/ui-design/references/motion",
		"skills/ui-design/references/anti-slop",
		"skills/ui-design/references/examples",
		"skills/ui-design/assets",
		"skills/ui-design/assets/examples",
		"skills/ui-design/assets/examples/shared",
		"skills/ui-design/assets/examples/approvals",
		"skills/ui-design/assets/examples/database",
		"skills/ui-design/assets/examples/databook",
		"skills/ui-design/assets/examples/office",
		"skills/ui-design/assets/examples/archive",
		"skills/ui-design/assets/examples/morrow",
		"skills/ui-design/assets/examples/booking",
		"skills/ui-design/assets/examples/canvas",
		"skills/ui-design/assets/examples/motion",
		"skills/ui-design/assets/examples/previews",
	} {
		for _, kind := range []string{"file", "symlink"} {
			t.Run(name+"/"+kind, func(t *testing.T) {
				home, target := t.TempDir(), t.TempDir()
				path := filepath.Join(home, ".meca", name)
				if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
					t.Fatal(err)
				}
				var err error
				if kind == "symlink" {
					err = os.Symlink(target, path)
				} else {
					err = os.WriteFile(path, []byte("preserve this file"), 0600)
				}
				if err != nil {
					t.Fatal(err)
				}
				if _, err := Initialize(home); err == nil {
					t.Fatal("accepted an unsafe skill directory")
				}
				entries, err := os.ReadDir(target)
				if err != nil || len(entries) != 0 {
					t.Fatalf("wrote through the skill link: %v %v", entries, err)
				}
				if kind == "file" {
					body, err := os.ReadFile(path)
					if err != nil || string(body) != "preserve this file" {
						t.Fatalf("replaced the existing file: %q %v", body, err)
					}
				}
			})
		}
	}
}
