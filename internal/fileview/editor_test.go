// Purpose: Protect complete-file editing from stale revisions, masked data and needless replacements.
// Pattern: Guarded filesystem integration tests.
// Usage: go test ./internal/fileview -run TestEditor.
// Related: internal/fileview/editor.go, internal/fileview/editor_write.go.
package fileview

import (
	"github.com/keonho-kim/meca/internal/policy"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEditorRevisionAndRedaction(t *testing.T) {
	base := t.TempDir()
	path := filepath.Join(base, "source.txt")
	if err := os.WriteFile(path, []byte("hello 한글\n"), 0640); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(base)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	reader := New(base, func(_ policy.Scope, p string) (Target, error) {
		return Target{Absolute: path, Relative: "source.txt", Root: root}, nil
	}, func(v string) string { return strings.ReplaceAll(v, "SECRET", "[redacted]") })
	settings := policy.Settings{Scope: policy.System}
	doc, err := reader.Edit(t.Context(), settings, "source.txt")
	if err != nil || !doc.Editable {
		t.Fatal(doc, err)
	}
	before, _ := os.Stat(path)
	if _, err := reader.Save(t.Context(), settings, "source.txt", doc.Text, doc.Revision); err != nil {
		t.Fatal(err)
	}
	after, _ := os.Stat(path)
	if !os.SameFile(before, after) {
		t.Fatal("unchanged save replaced the inode")
	}
	saved, err := reader.Save(t.Context(), settings, "source.txt", "changed\n", doc.Revision)
	if err != nil || saved.Revision == doc.Revision {
		t.Fatal(saved, err)
	}
	if _, err := reader.Save(t.Context(), settings, "source.txt", "stale overwrite", doc.Revision); err == nil {
		t.Fatal("stale revision accepted")
	}
	if _, err := reader.Save(t.Context(), settings, "source.txt", "SECRET", saved.Revision); err == nil {
		t.Fatal("credential persisted")
	}
	for _, value := range []string{"SECRET", "binary\x00text"} {
		if err := os.WriteFile(path, []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
		doc, err := reader.Edit(t.Context(), settings, "source.txt")
		if err != nil || doc.Editable {
			t.Fatal(doc, err)
		}
	}
}
