// Purpose: Verify the installed development library, reference reads and customized resources.
// Pattern: Installed skill contract tests through the real reader.
// Usage: Bootstrap and concurrent-initialization tests read the published development skill here.
// Related: internal/config/bootstrap/skills.go, internal/config/bootstrap/defaults/development, internal/skills/reader.go, internal/skills/resources.go.
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

func readDevelopmentSkill(t *testing.T, home string) map[string]int {
	t.Helper()
	r := skills.New(t.TempDir(), home, skills.Options{Redact: func(s string) string { return s }, Check: func(string) error { return nil }})
	page, err := r.List(t.Context(), "", 32)
	if err != nil {
		t.Fatal(err)
	}
	var id string
	for _, item := range page.Items {
		if item.Name == "development" && !item.ManualOnly {
			id = item.ID
		}
	}
	if id == "" {
		t.Fatal("development cannot be discovered automatically")
	}
	doc, err := r.Load(t.Context(), instruction.Selector{ID: id}, false)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Name != "development" || doc.Description == "" || doc.Body == "" || doc.Revision == "" || doc.ManualOnly {
		t.Fatal("incomplete development skill", doc.Metadata)
	}
	links := regexp.MustCompile(`\]\(([^)\s]+)\)`)
	depths := map[string]int{"SKILL.md": 0}
	queue := []string{"SKILL.md"}
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		body := readDevelopmentResource(t, r, doc.ID, home, name)
		if path.Ext(name) == ".md" && len(body) > 8<<10 {
			t.Fatalf("development instruction %s exceeds its 8KiB context budget", name)
		}
		for _, link := range links.FindAllStringSubmatch(body, -1) {
			target := strings.SplitN(link[1], "#", 2)[0]
			if target == "" || strings.Contains(target, "://") {
				continue
			}
			if path.IsAbs(target) {
				t.Fatalf("absolute reference in %s: %s", name, target)
			}
			target = path.Clean(path.Join(path.Dir(name), target))
			if target != "SKILL.md" && target != "NOTICE.md" && target != "LICENSE.txt" && !strings.HasPrefix(target, "references/") && !strings.HasPrefix(target, "assets/examples/") && !strings.HasPrefix(target, "scripts/") {
				t.Fatalf("instruction link leaves the reference tree: %s -> %s", name, target)
			}
			if _, exists := depths[target]; !exists {
				depths[target] = depths[name] + 1
				queue = append(queue, target)
			}
		}
	}
	for _, name := range []string{"LICENSE.txt", "NOTICE.md"} {
		readDevelopmentResource(t, r, doc.ID, home, name)
	}
	return depths
}

func readDevelopmentResource(t *testing.T, r *skills.Reader, id, home, name string) string {
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
	info, err := os.Lstat(filepath.Join(home, ".meca/skills/development", name))
	if err != nil || info.Mode().Perm() != 0600 || !info.Mode().IsRegular() {
		t.Fatal(name, info, err)
	}
	content := body.String()
	linkText := content
	if name == "assets/examples/shared/icons.svg" {
		// The SVG namespace identifies the vocabulary used by the sprite.
		linkText = strings.Replace(linkText, `xmlns="http://www.w3.org/2000/svg"`, "", 1)
	}
	if strings.HasPrefix(name, "assets/") && externalSkillLink.MatchString(linkText) {
		t.Fatalf("development resource %s contains an external link", name)
	}
	return content
}

func TestDevelopmentSkillReadsPagedExample(t *testing.T) {
	home := t.TempDir()
	if _, err := Initialize(t.Context(), home); err != nil {
		t.Fatal(err)
	}
	name := filepath.Join(home, ".meca/skills/development/assets/examples/shared/base.css")
	body := strings.Repeat(".example { color: inherit; } /* 한글 예시 */\n", 3000)
	if err := os.WriteFile(name, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	readDevelopmentSkill(t, home)
}

func TestBootstrapPublishesDevelopmentSkill(t *testing.T) {
	home := t.TempDir()
	if _, err := Initialize(t.Context(), home); err != nil {
		t.Fatal(err)
	}
	depths := readDevelopmentSkill(t, home)
	mentions := instruction.Mentions("Use #development")
	if len(mentions) != 1 || mentions[0] != "development" {
		t.Fatalf("development mention was not recognized: %v", mentions)
	}
	reader := skills.New(t.TempDir(), home, skills.Options{Redact: func(s string) string { return s }, Check: func(string) error { return nil }})
	document, err := reader.Load(t.Context(), instruction.Selector{ID: mentions[0]}, true)
	if err != nil || document.Name != "development" || document.Source != "meca" {
		t.Fatalf("development mention did not load the bundled skill: %+v %v", document, err)
	}
	err = fs.WalkDir(defaultSkillFiles, "defaults/development", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && path.Base(name) != "LICENSE.txt" && path.Base(name) != "NOTICE.md" {
			name = strings.TrimPrefix(name, "defaults/development/")
			depth, found := depths[name]
			if !found || depth > 3 {
				t.Errorf("resource %s is not discoverable within four levels: found=%v links=%d", name, found, depth)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

}

func TestBootstrapPreservesAndRestoresDevelopmentResources(t *testing.T) {
	home := t.TempDir()
	previous := filepath.Join(home, ".meca/skills/custom-review/SKILL.md")
	if err := os.MkdirAll(filepath.Dir(previous), 0700); err != nil {
		t.Fatal(err)
	}
	userSkill := []byte("---\nname: custom-review\ndescription: User-owned instructions.\n---\nKeep this skill.\n")
	if err := os.WriteFile(previous, userSkill, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Initialize(t.Context(), home); err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(home, ".meca/skills/development")
	custom := []byte("User-owned motion guidance.\n")
	path := filepath.Join(directory, "references/ui/motion.md")
	if err := os.WriteFile(path, custom, 0600); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(directory, "scripts/layout-check.js")
	customScript := []byte("// User-owned diagnostic.\n")
	if err := os.WriteFile(script, customScript, 0600); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(directory, "references/backend/storage.md")
	if err := os.Remove(missing); err != nil {
		t.Fatal(err)
	}
	missingScript := filepath.Join(directory, "scripts/motion-check.js")
	if err := os.Remove(missingScript); err != nil {
		t.Fatal(err)
	}
	if _, err := Initialize(t.Context(), home); err != nil {
		t.Fatal(err)
	}
	if body, err := os.ReadFile(path); err != nil || string(body) != string(custom) {
		t.Fatal("development resource edit was overwritten", err)
	}
	if body, err := os.ReadFile(script); err != nil || string(body) != string(customScript) {
		t.Fatal("development diagnostic edit was overwritten", err)
	}
	if body, err := os.ReadFile(missing); err != nil || len(body) == 0 {
		t.Fatal("missing development resource was not restored", err)
	}
	if body, err := os.ReadFile(missingScript); err != nil || len(body) == 0 {
		t.Fatal("missing development diagnostic was not restored", err)
	}
	if body, err := os.ReadFile(previous); err != nil || string(body) != string(userSkill) {
		t.Fatal("previous user-owned skill was changed", err)
	}
	readDevelopmentSkill(t, home)
}
