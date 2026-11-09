// Purpose: Verify installed backend-design discovery, bounded references and user-file preservation.
// Pattern: Embedded-bundle contract tests through the production skill reader.
// Usage: Bootstrap publication and concurrent-initialization tests read the new default bundle here.
// Related: internal/config/bootstrap/skills.go, internal/skills/reader.go, internal/skills/resources.go.
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

func readBackendSkill(t *testing.T, home string) {
	t.Helper()
	r := skills.New(t.TempDir(), home, skills.Options{Redact: func(s string) string { return s }, Check: func(string) error { return nil }})
	page, err := r.List(t.Context(), "", 32)
	if err != nil {
		t.Fatal(err)
	}
	var id string
	for _, item := range page.Items {
		if item.Name == "backend-design" && !item.ManualOnly {
			id = item.ID
		}
	}
	if id == "" {
		t.Fatal("backend-design cannot be discovered automatically")
	}
	doc, err := r.Load(t.Context(), instruction.Selector{ID: id}, false)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Name != "backend-design" || doc.Description == "" || doc.Body == "" || doc.Revision == "" || doc.ManualOnly {
		t.Fatal("incomplete backend-design skill", doc.Metadata)
	}
	links := regexp.MustCompile(`\]\(([^)\s]+)\)`)
	depths := map[string]int{"SKILL.md": 0}
	queue := []string{"SKILL.md"}
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		resource, err := r.ReadFile(t.Context(), id, name, 0, skills.MaxResourceBytes, "")
		if err != nil {
			t.Fatalf("unreadable backend-design resource %s: %v", name, err)
		}
		if resource.Binary || resource.Text == "" || resource.Revision == "" || resource.NextOffset != 0 || resource.TotalBytes > 8<<10 {
			t.Fatalf("invalid or oversized backend-design resource %s: %+v", name, resource)
		}
		info, err := os.Lstat(filepath.Join(home, ".meca/skills/backend-design", name))
		if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 {
			t.Fatal(name, info, err)
		}
		for _, link := range links.FindAllStringSubmatch(resource.Text, -1) {
			target := strings.SplitN(link[1], "#", 2)[0]
			if strings.Contains(target, "://") {
				continue
			}
			if path.IsAbs(target) {
				t.Fatalf("absolute skill resource link: %s -> %s", name, target)
			}
			target = path.Clean(path.Join(path.Dir(name), target))
			if target != "SKILL.md" && !strings.HasPrefix(target, "references/") {
				t.Fatalf("resource link leaves backend-design: %s -> %s", name, target)
			}
			if _, found := depths[target]; !found {
				depths[target] = depths[name] + 1
				queue = append(queue, target)
			}
		}
	}
	err = fs.WalkDir(defaultSkillFiles, "defaults/backend-design", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		name = strings.TrimPrefix(name, "defaults/backend-design/")
		if depth, found := depths[name]; !found || depth > 2 {
			t.Errorf("backend-design resource %s is not reachable within two link hops", name)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestBootstrapPublishesBackendSkill(t *testing.T) {
	home := t.TempDir()
	if _, err := Initialize(home); err != nil {
		t.Fatal(err)
	}
	readBackendSkill(t, home)
}

func TestBootstrapPreservesAndRestoresBackendReferences(t *testing.T) {
	home := t.TempDir()
	if _, err := Initialize(home); err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(home, ".meca/skills/backend-design")
	customPath := filepath.Join(directory, "references/operations/diagnostics.md")
	custom := []byte("User-owned backend diagnostics guidance.\n")
	if err := os.WriteFile(customPath, custom, 0600); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(directory, "references/storage/postgres-queries.md")
	if err := os.Remove(missing); err != nil {
		t.Fatal(err)
	}
	if _, err := Initialize(home); err != nil {
		t.Fatal(err)
	}
	if body, err := os.ReadFile(customPath); err != nil || string(body) != string(custom) {
		t.Fatal("backend-design user edit was overwritten", err)
	}
	if body, err := os.ReadFile(missing); err != nil || len(body) == 0 {
		t.Fatal("missing backend-design reference was not restored", err)
	}
	readBackendSkill(t, home)
}
