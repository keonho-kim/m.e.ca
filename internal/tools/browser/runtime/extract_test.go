// Purpose: Reject escaping archives and preserve executable files with safe framework links.
// Pattern: Small ZIP fixtures with an isolated extraction root.
// Usage: go test ./internal/tools/browser/runtime -run TestExtract.
// Related: internal/tools/browser/runtime/extract.go.
package runtime

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractPathsAndLinks(t *testing.T) {
	for _, name := range []string{"../escape", "/absolute", "ok/../../escape", "..\\escape"} {
		t.Run(name, func(t *testing.T) {
			path := zipFixture(t, []zipEntry{{name, "bad", 0600}})
			if err := extract(t.Context(), path, t.TempDir()); err == nil {
				t.Fatal("unsafe archive accepted")
			}
		})
	}
	root := t.TempDir()
	path := zipFixture(t, []zipEntry{{"app/binary", "hello", 0700}, {"app/current", "binary", os.ModeSymlink | 0700}})
	if err := extract(t.Context(), path, root); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "app/current"))
	if err != nil || string(data) != "hello" {
		t.Fatal(err)
	}
	if !executable(filepath.Join(root, "app/binary")) {
		t.Fatal("lost executable mode")
	}
	bad := zipFixture(t, []zipEntry{{"app/link", "../../outside", os.ModeSymlink | 0700}})
	if err := extract(t.Context(), bad, t.TempDir()); err == nil {
		t.Fatal("escaping link accepted")
	}
}

type zipEntry struct {
	name, body string
	mode       os.FileMode
}

func zipFixture(t *testing.T, entries []zipEntry) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "runtime.zip")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	for _, entry := range entries {
		header := &zip.FileHeader{Name: entry.name}
		header.SetMode(entry.mode)
		out, err := writer.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := out.Write([]byte(entry.body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}
