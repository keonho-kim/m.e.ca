// Purpose: Verify bounded ui-design discovery, installed code assets and preserved user edits.
// Pattern: Installed skill contract tests through the real reader.
// Usage: Bootstrap and concurrent-initialization tests read the published ui-design skill here.
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

func readUIDesignSkill(t *testing.T, home string) map[string]int {
	t.Helper()
	r := skills.New(t.TempDir(), home, skills.Options{Redact: func(s string) string { return s }, Check: func(string) error { return nil }})
	page, err := r.List(t.Context(), "", 32)
	if err != nil {
		t.Fatal(err)
	}
	var id string
	for _, item := range page.Items {
		if item.Name == "ui-design" && !item.ManualOnly {
			id = item.ID
		}
	}
	if id == "" {
		t.Fatal("ui-design cannot be discovered automatically")
	}
	doc, err := r.Load(t.Context(), instruction.Selector{ID: id}, false)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Name != "ui-design" || doc.Description == "" || doc.Body == "" || doc.Revision == "" || doc.ManualOnly {
		t.Fatal("incomplete ui-design skill", doc.Metadata)
	}
	links := regexp.MustCompile(`\]\(([^)\s]+\.(?:md|html|css|js|json|svg|png|jpg|webp|txt)(?:#[^)\s]+)?)\)`)
	depths := map[string]int{"SKILL.md": 0}
	queue := []string{"SKILL.md"}
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		body := readUIDesignResource(t, r, doc.ID, home, name)
		if !strings.HasPrefix(name, "assets/") && len(body) > 8<<10 {
			t.Fatalf("ui-design instruction %s exceeds its 8KiB context budget", name)
		}
		for _, link := range links.FindAllStringSubmatch(body, -1) {
			target := strings.SplitN(link[1], "#", 2)[0]
			if strings.Contains(target, "://") {
				continue
			}
			target = path.Clean(path.Join(path.Dir(name), target))
			if target != "SKILL.md" && !strings.HasPrefix(target, "references/") && !strings.HasPrefix(target, "assets/examples/") {
				t.Fatalf("instruction link leaves the reference tree: %s -> %s", name, target)
			}
			if _, exists := depths[target]; !exists {
				depths[target] = depths[name] + 1
				queue = append(queue, target)
			}
		}
	}
	for _, name := range []string{"LICENSE.txt", "NOTICE.md"} {
		readUIDesignResource(t, r, doc.ID, home, name)
	}
	return depths
}

func readUIDesignResource(t *testing.T, r *skills.Reader, id, home, name string) string {
	t.Helper()
	var body strings.Builder
	var offset, total int64
	var revision string
	for {
		page, err := r.ReadFile(t.Context(), id, name, offset, skills.MaxResourceBytes, revision)
		if err != nil || page.TotalBytes < 1 || page.TotalBytes > 1<<20 || offset > 0 && page.TotalBytes != total {
			t.Fatalf("bundled resource %s is unreadable or exceeds 1MiB: %+v %v", name, page, err)
		}
		total = page.TotalBytes
		revision = page.Revision
		if page.Binary {
			media := map[string]string{".png": "image/png", ".jpg": "image/jpeg", ".webp": "image/webp"}
			if offset != 0 || page.NextOffset != 0 || media[path.Ext(name)] != page.MediaType {
				t.Fatalf("unexpected binary resource %s: %+v", name, page)
			}
			break
		}
		if page.Text == "" {
			t.Fatalf("empty text page in %s at %d", name, offset)
		}
		body.WriteString(page.Text)
		if page.NextOffset == 0 {
			break
		}
		if page.NextOffset <= offset || page.NextOffset >= total {
			t.Fatalf("resource %s made no paging progress: %+v", name, page)
		}
		offset = page.NextOffset
	}
	info, err := os.Lstat(filepath.Join(home, ".meca/skills/ui-design", name))
	if err != nil || info.Mode().Perm() != 0600 || !info.Mode().IsRegular() {
		t.Fatal(name, info, err)
	}
	return body.String()
}

func TestUIDesignSkillReadsPagedExample(t *testing.T) {
	home := t.TempDir()
	if _, err := Initialize(home); err != nil {
		t.Fatal(err)
	}
	name := filepath.Join(home, ".meca/skills/ui-design/assets/examples/shared/base.css")
	body := strings.Repeat(".example { color: inherit; } /* 한글 예시 */\n", 3000)
	if err := os.WriteFile(name, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	readUIDesignSkill(t, home)
}

func TestBootstrapPublishesUIDesignSkill(t *testing.T) {
	home := t.TempDir()
	if _, err := Initialize(home); err != nil {
		t.Fatal(err)
	}
	depths := readUIDesignSkill(t, home)
	mentions := instruction.Mentions("Use #ui-design")
	if len(mentions) != 1 || mentions[0] != "ui-design" {
		t.Fatalf("ui-design mention was not recognized: %v", mentions)
	}
	reader := skills.New(t.TempDir(), home, skills.Options{Redact: func(s string) string { return s }, Check: func(string) error { return nil }})
	document, err := reader.Load(t.Context(), instruction.Selector{ID: mentions[0]}, true)
	if err != nil || document.Name != "ui-design" || document.Source != "meca" {
		t.Fatalf("ui-design mention did not load the bundled skill: %+v %v", document, err)
	}
	err = fs.WalkDir(defaultSkillFiles, "defaults/ui-design", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && path.Base(name) != "LICENSE.txt" && path.Base(name) != "NOTICE.md" {
			name = strings.TrimPrefix(name, "defaults/ui-design/")
			depth, found := depths[name]
			if !found || depth > 2 {
				t.Errorf("resource %s is not discoverable within three levels: found=%v links=%d", name, found, depth)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"frontend-essentials", "application-design"} {
		if _, err := os.Lstat(filepath.Join(home, ".meca/skills", name)); !os.IsNotExist(err) {
			t.Fatalf("bootstrap still publishes replaced default skill %s: %v", name, err)
		}
	}
}

func TestBootstrapPreservesAndRestoresUIDesignResources(t *testing.T) {
	home := t.TempDir()
	previous := filepath.Join(home, ".meca/skills/application-design/SKILL.md")
	if err := os.MkdirAll(filepath.Dir(previous), 0700); err != nil {
		t.Fatal(err)
	}
	userSkill := []byte("---\nname: application-design\ndescription: User-owned instructions.\n---\nKeep this skill.\n")
	if err := os.WriteFile(previous, userSkill, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Initialize(home); err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(home, ".meca/skills/ui-design")
	custom := []byte("User-owned motion guidance.\n")
	path := filepath.Join(directory, "references/motion.md")
	if err := os.WriteFile(path, custom, 0600); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(directory, "references/layout-map.md")
	if err := os.Remove(missing); err != nil {
		t.Fatal(err)
	}
	if _, err := Initialize(home); err != nil {
		t.Fatal(err)
	}
	if body, err := os.ReadFile(path); err != nil || string(body) != string(custom) {
		t.Fatal("ui-design resource edit was overwritten", err)
	}
	if body, err := os.ReadFile(missing); err != nil || len(body) == 0 {
		t.Fatal("missing ui-design resource was not restored", err)
	}
	if body, err := os.ReadFile(previous); err != nil || string(body) != string(userSkill) {
		t.Fatal("previous user-owned skill was changed", err)
	}
	readUIDesignSkill(t, home)
}
