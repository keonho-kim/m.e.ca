// Purpose: Verify bounded application-design discovery, installed code assets and preserved user edits.
// Pattern: Installed skill contract tests through the real reader.
// Usage: Bootstrap and concurrent-initialization tests read the published application-design skill here.
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

func readApplicationSkill(t *testing.T, home string) map[string]int {
	t.Helper()
	r := skills.New(t.TempDir(), home, skills.Options{Redact: func(s string) string { return s }, Check: func(string) error { return nil }})
	page, err := r.List(t.Context(), "", 32)
	if err != nil {
		t.Fatal(err)
	}
	var id string
	for _, item := range page.Items {
		if item.Name == "application-design" && !item.ManualOnly {
			id = item.ID
		}
	}
	if id == "" {
		t.Fatal("application-design cannot be discovered automatically")
	}
	doc, err := r.Load(t.Context(), instruction.Selector{ID: id}, false)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Name != "application-design" || doc.Description == "" || doc.Body == "" || doc.Revision == "" || doc.ManualOnly {
		t.Fatal("incomplete application-design skill", doc.Metadata)
	}
	links := regexp.MustCompile(`\]\(([^)\s]+\.(?:md|html|css|js|json|svg|png|jpg|webp|txt)(?:#[^)\s]+)?)\)`)
	depths := map[string]int{"SKILL.md": 0}
	queue := []string{"SKILL.md"}
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		body := readApplicationResource(t, r, doc.ID, home, name)
		if !strings.HasPrefix(name, "assets/") && len(body) > 8<<10 {
			t.Fatalf("application-design instruction %s exceeds its 8KiB context budget", name)
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
		readApplicationResource(t, r, doc.ID, home, name)
	}
	return depths
}

func readApplicationResource(t *testing.T, r *skills.Reader, id, home, name string) string {
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
	info, err := os.Lstat(filepath.Join(home, ".meca/skills/application-design", name))
	if err != nil || info.Mode().Perm() != 0600 || !info.Mode().IsRegular() {
		t.Fatal(name, info, err)
	}
	return body.String()
}

func TestApplicationSkillReadsPagedExample(t *testing.T) {
	home := t.TempDir()
	if _, err := Initialize(home); err != nil {
		t.Fatal(err)
	}
	name := filepath.Join(home, ".meca/skills/application-design/assets/examples/shared/base.css")
	body := strings.Repeat(".example { color: inherit; } /* 한글 예시 */\n", 3000)
	if err := os.WriteFile(name, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	readApplicationSkill(t, home)
}

func TestBootstrapPublishesApplicationSkill(t *testing.T) {
	home := t.TempDir()
	if _, err := Initialize(home); err != nil {
		t.Fatal(err)
	}
	depths := readApplicationSkill(t, home)
	err := fs.WalkDir(defaultSkillFiles, "defaults/application-design", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && path.Base(name) != "LICENSE.txt" && path.Base(name) != "NOTICE.md" {
			name = strings.TrimPrefix(name, "defaults/application-design/")
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
	if _, err := os.Lstat(filepath.Join(home, ".meca/skills/frontend-essentials")); !os.IsNotExist(err) {
		t.Fatal("bootstrap still publishes the replaced default skill", err)
	}
}

func TestBootstrapPreservesAndRestoresApplicationResources(t *testing.T) {
	home := t.TempDir()
	if _, err := Initialize(home); err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(home, ".meca/skills/application-design")
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
		t.Fatal("application-design resource edit was overwritten", err)
	}
	if body, err := os.ReadFile(missing); err != nil || len(body) == 0 {
		t.Fatal("missing application-design resource was not restored", err)
	}
	readApplicationSkill(t, home)
}
