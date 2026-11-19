// Purpose: Verify default skill publication, privacy and preservation during initialization.
// Pattern: Filesystem contract tests.
// Usage: go test ./internal/config/bootstrap -run 'TestBootstrap.*Skill|TestConcurrentBootstrap'.
// Related: internal/config/bootstrap/skills.go, internal/config/bootstrap/bootstrap.go, internal/skills/reader.go.
package bootstrap

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"testing"

	"github.com/keonho-kim/meca/internal/config/statefs"
	"github.com/keonho-kim/meca/internal/skills"
	"github.com/keonho-kim/meca/internal/skills/instruction"
)

var externalSkillLink = regexp.MustCompile(`(?i)\b(?:https?://|ftp://|mailto:|tel:|www\.)|\]\(\s*<?//|(?:href|src|srcset)\s*=\s*["']//|url\(\s*["']?//`)

type skillLeaseWait struct {
	context.Context
	waiting chan struct{}
	once    sync.Once
}

func (c *skillLeaseWait) Done() <-chan struct{} {
	c.once.Do(func() { close(c.waiting) })
	return c.Context.Done()
}

func TestBootstrapRejectsSkillDirectoryReplacedWhileWaiting(t *testing.T) {
	root, err := statefs.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	skills, _, err := openPrivateDirectory(root, "skills")
	if err != nil {
		t.Fatal(err)
	}
	defer skills.Close()
	lease, err := statefs.Lease(skills, true)
	if err != nil {
		t.Fatal(err)
	}
	release := sync.OnceValue(lease.Close)
	defer release()
	ctx, cancel := context.WithCancel(t.Context())
	wait := &skillLeaseWait{Context: ctx, waiting: make(chan struct{})}
	finished := make(chan struct{})
	var bootErr error
	go func() {
		bootErr = bootstrapSkills(wait, root)
		close(finished)
	}()
	defer func() {
		cancel()
		<-finished
	}()
	select {
	case <-wait.waiting:
	case <-finished:
		t.Fatalf("bootstrap did not wait for the held lease: %v", bootErr)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	if err := root.Rename("skills", "previous"); err != nil {
		t.Fatal(err)
	}
	if err := root.Mkdir("skills", 0700); err != nil {
		t.Fatal(err)
	}
	if err := release(); err != nil {
		t.Fatal(err)
	}
	<-finished
	if bootErr == nil {
		t.Fatal("bootstrap accepted a directory replaced while it waited")
	}
	for _, name := range []string{"skills", "previous"} {
		entries, err := os.ReadDir(filepath.Join(root.Name(), name))
		if err != nil || len(entries) != 0 {
			t.Fatalf("published into %s after its identity changed: %v %v", name, entries, err)
		}
	}
}

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
	if _, err := Initialize(t.Context(), home); err != nil {
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
	if _, err := Initialize(t.Context(), home); err != nil {
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
	if _, err := Initialize(t.Context(), home); err != nil {
		t.Fatal(err)
	}
	if body, err := os.ReadFile(manifest); err != nil || string(body) != custom {
		t.Fatalf("custom skill overwritten: %q %v", body, err)
	}
	if err := os.Remove(manifest); err != nil {
		t.Fatal(err)
	}
	if _, err := Initialize(t.Context(), home); err != nil {
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
				if _, err := Initialize(t.Context(), home); err == nil {
					t.Fatal("accepted an unsafe skill directory")
				}
				if name != "skills" {
					installed := filepath.Join(home, ".meca/skills/backend-design/SKILL.md")
					if _, err := os.Lstat(installed); !os.IsNotExist(err) {
						t.Fatalf("published a sibling bundle before rejecting the conflicting directory: %v", err)
					}
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
