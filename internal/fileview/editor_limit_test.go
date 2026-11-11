// Purpose: Enforce the complete UTF-8 editor boundary without changing files on rejected saves.
// Pattern: Filesystem boundary regression tests.
// Usage: go test ./internal/fileview -run TestEditorTwoMiB.
// Related: internal/fileview/editor.go, internal/fileview/editor_write.go.
package fileview

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/policy"
)

func TestEditorTwoMiB(t *testing.T) {
	base := t.TempDir()
	path := filepath.Join(base, "document.txt")
	if err := os.WriteFile(path, []byte("initial"), 0600); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(base)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = root.Close() })
	reader := New(base, func(policy.Scope, string) (Target, error) {
		return Target{Absolute: path, Relative: "document.txt", Root: root}, nil
	}, func(value string) string { return value })
	settings := policy.Settings{Scope: policy.System}
	doc, err := reader.Edit(t.Context(), settings, "document.txt")
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Repeat("한", (2<<20)/3) + "xx"
	saved, err := reader.Save(t.Context(), settings, "document.txt", text, doc.Revision)
	if err != nil {
		t.Fatalf("2MiB UTF-8 save rejected: %v", err)
	}
	opened, err := reader.Edit(t.Context(), settings, "document.txt")
	if err != nil || !opened.Editable || opened.Text != text || opened.Revision != saved.Revision {
		t.Fatalf("2MiB read mismatch: editable=%v bytes=%d error=%v", opened.Editable, len(opened.Text), err)
	}
	if _, err := reader.Save(t.Context(), settings, "document.txt", text+"x", saved.Revision); err == nil {
		t.Fatal("oversized save accepted")
	}
	body, err := os.ReadFile(path)
	if err != nil || string(body) != text {
		t.Fatal("rejected save changed the document")
	}
	if err := os.WriteFile(path, []byte(text+"x"), 0600); err != nil {
		t.Fatal(err)
	}
	opened, err = reader.Edit(t.Context(), settings, "document.txt")
	if err != nil || opened.Editable || opened.Text != "" {
		t.Fatalf("oversized file did not retain paged read-only access: %v", err)
	}
}
