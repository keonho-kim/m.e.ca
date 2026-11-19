// Purpose: Verify application guide discovery, local navigation and the per-type bundle layout.
// Pattern: Installed skill contract test through the production reader.
// Usage: go test ./internal/config/bootstrap -run TestBootstrapPublishesApplicationSkill.
// Related: internal/config/bootstrap/skills.go, internal/config/bootstrap/defaults/application-best-dev-practices/SKILL.md, internal/skills/reader.go.
package bootstrap

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/skills"
	"github.com/keonho-kim/meca/internal/skills/instruction"
)

func TestBootstrapPublishesApplicationSkill(t *testing.T) {
	const skill = "application-best-dev-practices"
	home := t.TempDir()
	if _, err := Initialize(t.Context(), home); err != nil {
		t.Fatal(err)
	}
	reader := skills.New(t.TempDir(), home, skills.Options{Redact: func(s string) string { return s }, Check: func(string) error { return nil }})
	catalog, err := reader.List(t.Context(), "", 32)
	if err != nil {
		t.Fatal(err)
	}
	var id string
	for _, item := range catalog.Items {
		if item.Name == skill && !item.ManualOnly {
			id = item.ID
			break
		}
	}
	if id == "" {
		t.Fatal("application skill is missing from automatic discovery")
	}
	mentions := instruction.Mentions("Use #" + skill)
	if len(mentions) != 1 || mentions[0] != skill {
		t.Fatalf("application skill mention was not recognized: %v", mentions)
	}
	document, err := reader.Load(t.Context(), instruction.Selector{ID: mentions[0]}, true)
	if err != nil || document.ID != id || document.Source != "meca" || document.Body == "" {
		t.Fatalf("cannot load the installed application skill: %v", err)
	}

	read := func(name string) string {
		t.Helper()
		resource, err := reader.ReadFile(t.Context(), id, name, 0, skills.MaxResourceBytes, "")
		if err != nil {
			t.Fatalf("cannot read %s: %v", name, err)
		}
		if resource.Binary || resource.Text == "" || resource.NextOffset != 0 || resource.TotalBytes > 8<<10 {
			t.Fatalf("invalid or oversized application guide %s", name)
		}
		info, err := os.Lstat(filepath.Join(home, ".meca", "skills", skill, name))
		if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 {
			t.Fatalf("application guide %s has invalid publication metadata: %v %v", name, info, err)
		}
		if externalSkillLink.MatchString(resource.Text) {
			t.Fatalf("application guide %s contains an external link", name)
		}
		return resource.Text
	}
	resources := map[string]string{"SKILL.md": read("SKILL.md")}
	references := path.Join("defaults", skill, "references")
	directories, err := fs.ReadDir(defaultSkillFiles, references)
	if err != nil || len(directories) < 18 {
		t.Fatalf("application skill lacks the app type folders: %v", err)
	}
	for _, entry := range directories {
		dir := path.Join("references", entry.Name())
		files, err := fs.ReadDir(defaultSkillFiles, path.Join(references, entry.Name()))
		if err != nil || len(files) != 3 {
			t.Fatalf("application folder %s needs three guide files: %v", dir, err)
		}
		guide := path.Join(dir, "app-guide.md")
		if !strings.Contains(resources["SKILL.md"], "]("+guide+")") {
			t.Fatalf("application guide %s is missing from the entrypoint", guide)
		}
		resources[guide] = read(guide)
		for _, name := range []string{"do-and-dont.md", "implementation-qa.md"} {
			if !strings.Contains(resources[guide], "]("+name+")") {
				t.Fatalf("application guide %s does not link to %s", guide, name)
			}
			resource := path.Join(dir, name)
			resources[resource] = read(resource)
		}
	}

	links := regexp.MustCompile(`\]\(([^)\s]+)\)`)
	for name, content := range resources {
		for _, link := range links.FindAllStringSubmatch(content, -1) {
			target, _, _ := strings.Cut(link[1], "#")
			if target == "" {
				continue
			}
			destination := path.Clean(path.Join(path.Dir(name), target))
			if _, found := resources[destination]; path.IsAbs(target) || !found {
				t.Errorf("application guide %s links outside the bundle: %s", name, target)
			}
		}
	}
}
