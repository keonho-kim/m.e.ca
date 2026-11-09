// Purpose: Verify portable skill discovery, explicit reads and confined reference access.
// Pattern: Filesystem contract tests.
// Usage: go test ./internal/skills.
// Related: internal/skills/reader.go, internal/skills/catalog.go, internal/skills/format.go.
package skills

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/redaction"
	"github.com/keonho-kim/meca/internal/skills/instruction"
)

func skillFixture(t *testing.T) (*Reader, string, string) {
	t.Helper()
	project, home := t.TempDir(), t.TempDir()
	return New(project, home, Options{Redact: func(s string) string { return strings.ReplaceAll(s, "secret-value", "[redacted]") }, Check: func(string) error { return nil }}), project, home
}

func writeSkill(t *testing.T, root, name, description, body string) string {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: "+name+"\ndescription: "+description+"\n---\n"+body), 0600); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestDiscoveryPriorityAndPortableReads(t *testing.T) {
	r, project, home := skillFixture(t)
	for _, root := range []string{filepath.Join(project, ".agents/skills"), filepath.Join(home, ".meca/skills"), filepath.Join(home, ".agents/skills")} {
		writeSkill(t, root, "review", "Review source", root+" secret-value")
	}
	page, err := r.List(context.Background(), "", 32)
	if err != nil || len(page.Items) != 3 {
		t.Fatalf("page=%+v err=%v", page, err)
	}
	doc, err := r.Load(context.Background(), instruction.Selector{ID: "review"}, true)
	if err != nil || doc.Source != "project" || strings.Contains(doc.Body, "secret-value") {
		t.Fatalf("doc=%+v err=%v", doc, err)
	}
	global, err := r.Load(context.Background(), instruction.Selector{ID: "meca:review"}, false)
	if err != nil || global.Source != "meca" || global.Revision == "" {
		t.Fatalf("doc=%+v err=%v", global, err)
	}
}

func TestInvalidSkillDoesNotHideValidSkills(t *testing.T) {
	r, project, _ := skillFixture(t)
	root := filepath.Join(project, ".agents/skills")
	writeSkill(t, root, "good", "Good skill", "Instructions")
	bad := writeSkill(t, root, "bad", "Bad skill", "Instructions")
	if err := os.WriteFile(filepath.Join(bad, "SKILL.md"), []byte("---\nname: bad\nname: bad\ndescription: duplicate\n---\nbody"), 0600); err != nil {
		t.Fatal(err)
	}
	page, err := r.List(context.Background(), "", 32)
	if err != nil || len(page.Items) != 1 || len(page.Diagnostics) != 1 {
		t.Fatalf("page=%+v err=%v", page, err)
	}
	if _, err := r.Load(context.Background(), instruction.Selector{ID: "project:bad"}, true); err == nil {
		t.Fatal("invalid explicitly selected skill accepted")
	}
}

func TestReferencesRejectEscapeAndRemainBounded(t *testing.T) {
	r, project, _ := skillFixture(t)
	dir := writeSkill(t, filepath.Join(project, ".agents/skills"), "read", "Read docs", "Use references")
	if err := os.WriteFile(filepath.Join(dir, "note.txt"), []byte(strings.Repeat("x", MaxResourceBytes+5)), 0600); err != nil {
		t.Fatal(err)
	}
	page, err := r.ReadFile(context.Background(), "project:read", "note.txt", 0, MaxResourceBytes, "")
	if err != nil || len(page.Text) != MaxResourceBytes || page.NextOffset != MaxResourceBytes {
		t.Fatalf("page=%+v err=%v", page, err)
	}
	outside := filepath.Join(t.TempDir(), "private.txt")
	if err := os.WriteFile(outside, []byte("never disclose"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "escape")); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"../SKILL.md", "escape", outside} {
		if _, err := r.ReadFile(context.Background(), "project:read", path, 0, 100, ""); err == nil {
			t.Fatalf("accepted %q", path)
		}
	}
}

func TestManualOnlyAndSelectorValidation(t *testing.T) {
	r, project, _ := skillFixture(t)
	dir := writeSkill(t, filepath.Join(project, ".agents/skills"), "manual", "Manual only", "Instructions")
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: manual\ndescription: Manual only\ndisable-model-invocation: true\n---\nInstructions"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Load(context.Background(), instruction.Selector{ID: "manual"}, false); err == nil {
		t.Fatal("implicit manual-only load accepted")
	}
	if _, err := r.Load(context.Background(), instruction.Selector{Path: dir}, true); err != nil {
		t.Fatal(err)
	}
	for _, s := range []instruction.Selector{{}, {ID: "manual", Path: dir}, {ID: "project:../manual"}} {
		if _, err := r.Load(context.Background(), s, true); err == nil {
			t.Fatalf("accepted %+v", s)
		}
	}
}

func TestPathLoadedReferencesAndCanceledEmptyCatalog(t *testing.T) {
	r, _, _ := skillFixture(t)
	dir := writeSkill(t, t.TempDir(), "outside", "External instructions", "Use reference.txt")
	if err := os.WriteFile(filepath.Join(dir, "reference.txt"), []byte("external-reference"), 0600); err != nil {
		t.Fatal(err)
	}
	doc, err := r.Load(context.Background(), instruction.Selector{Path: dir}, true)
	if err != nil {
		t.Fatal(err)
	}
	file, err := r.ReadFile(context.Background(), doc.ID, "reference.txt", 0, 100, "")
	if err != nil || file.Text != "external-reference" {
		t.Fatal(file, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := r.List(ctx, "", 32); err == nil {
		t.Fatal("canceled empty discovery returned success")
	}
}

func TestDiscoveryDeduplicatesDirectoriesButPreservesResourceBases(t *testing.T) {
	r, project, home := skillFixture(t)
	projectRoot, globalRoot := filepath.Join(project, ".agents/skills"), filepath.Join(home, ".meca/skills")
	dir := writeSkill(t, projectRoot, "same", "Same description", "Same body")
	writeSkill(t, globalRoot, "same", "Same description", "Same body")
	if err := os.Symlink(dir, filepath.Join(globalRoot, "alias")); err != nil {
		t.Fatal(err)
	}
	page, err := r.List(context.Background(), "", 32)
	if err != nil || len(page.Items) != 2 {
		t.Fatalf("items=%+v err=%v", page.Items, err)
	}
}

func TestReferencePageRedactsCredentialsAcrossBothEdges(t *testing.T) {
	_, project, home := skillFixture(t)
	var registry redaction.Registry
	if err := registry.Remember("credential-canary"); err != nil {
		t.Fatal(err)
	}
	r := New(project, home, Options{Redact: registry.Redact, RedactRange: registry.RedactRange, Check: func(string) error { return nil }})
	dir := writeSkill(t, filepath.Join(project, ".agents/skills"), "pages", "Paged reference", "Read note.txt")
	if err := os.WriteFile(filepath.Join(dir, "note.txt"), []byte("AAAAAAcredential-canaryBBBB"), 0600); err != nil {
		t.Fatal(err)
	}
	revision := ""
	for offset := int64(0); ; {
		page, err := r.ReadFile(context.Background(), "project:pages", "note.txt", offset, 10, revision)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(page.Text, "cred") || strings.Contains(page.Text, "canary") {
			t.Fatalf("credential page leaked: %q", page.Text)
		}
		revision = page.Revision
		if page.NextOffset == 0 {
			break
		}
		if page.NextOffset <= offset {
			t.Fatal("page did not advance")
		}
		offset = page.NextOffset
	}
}
