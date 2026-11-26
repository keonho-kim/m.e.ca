// Purpose: Verify the installed development search command finds current local references.
// Pattern: Command integration tests over a disposable installed skill.
// Usage: go test ./internal/config/bootstrap -run TestDevelopmentSearch.
// Related: internal/config/bootstrap/skills.go, internal/config/bootstrap/defaults/development/scripts/search.sh.
package bootstrap

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDevelopmentSearch(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home with spaces")
	if err := os.MkdirAll(home, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := Initialize(t.Context(), home); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(home, ".meca", "skills", "development")
	script := filepath.Join(root, "scripts", "search.sh")
	work := t.TempDir()
	search := func(t *testing.T, query string) string {
		t.Helper()
		command := exec.CommandContext(t.Context(), "sh", script, query)
		command.Dir = work
		output, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("search %q: %v\n%s", query, err, output)
		}
		return string(output)
	}
	for _, tc := range []struct {
		query, path, heading string
	}{
		{"prefix suggestions", "lookup-and-updates.md", "Prefix suggestions over a known dictionary"},
		{"delayed tasks", "scheduling-and-streams.md", "Delayed tasks and expiring entries"},
		{"incremental parsing", "ranges-and-edits.md", "Incrementally parsing or highlighting changed text"},
	} {
		t.Run(tc.query, func(t *testing.T) {
			output := search(t, tc.query)
			if !strings.Contains(output, tc.path) || !strings.Contains(output, tc.heading) {
				t.Fatalf("missing relevant reference: %s", output)
			}
			if len(strings.Split(strings.TrimSpace(output), "\n")) > 10 {
				t.Fatalf("unbounded search results: %s", output)
			}
		})
	}

	t.Run("current content and literal queries", func(t *testing.T) {
		path := filepath.Join(root, "references", "local notes.md")
		body := "# Local notes\n\n## Literal query\n\nskill.literal[0].needle $(touch search-side-effect)\n"
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
		output := search(t, "skill.literal[0].needle $(touch search-side-effect)")
		if !strings.Contains(output, "references/local notes.md:3: Literal query") {
			t.Fatalf("literal query did not find the installed edit: %s", output)
		}
		if _, err := os.Stat(filepath.Join(work, "search-side-effect")); !os.IsNotExist(err) {
			t.Fatalf("query was interpreted as a command: %v", err)
		}
		if err := os.WriteFile(path, []byte("# Updated lookup\n\nupdated-local-reference-marker\n"), 0600); err != nil {
			t.Fatal(err)
		}
		if output := search(t, "updated-local-reference-marker"); !strings.Contains(output, "references/local notes.md:1: Updated lookup") {
			t.Fatalf("search missed a changed reference: %s", output)
		}
	})

	t.Run("search stays in the reference tree", func(t *testing.T) {
		outside := filepath.Join(work, "outside.md")
		if err := os.WriteFile(outside, []byte("# Outside\n\noutside-reference-only-marker\n"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, filepath.Join(root, "references", "external.md")); err != nil {
			t.Fatal(err)
		}
		if output := search(t, "outside-reference-only-marker"); !strings.Contains(output, "No matching reference") {
			t.Fatalf("search followed a reference symlink: %s", output)
		}
	})

	t.Run("query is required", func(t *testing.T) {
		command := exec.CommandContext(t.Context(), "sh", script, " \t ")
		output, err := command.CombinedOutput()
		if err == nil || !strings.Contains(string(output), "Usage") {
			t.Fatalf("empty query was accepted: %v\n%s", err, output)
		}
	})
}
