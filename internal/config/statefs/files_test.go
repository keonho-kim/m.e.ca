// Purpose: Verify batch publication preserves existing and concurrently created files.
// Pattern: Deterministic filesystem contract test.
// Usage: go test ./internal/config/statefs -run TestPublishMissingFiles.
// Related: internal/config/statefs/files.go, internal/config/bootstrap/skills.go.
package statefs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPublishMissingFilesPreservesExistingAndRacingWriters(t *testing.T) {
	dir := t.TempDir()
	root, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	writeUserFile := func(name string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte("user content"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	writeUserFile("custom.md")
	err = PublishMissingFiles(root, []string{"custom.md", "default.md", "racing.md"}, func(name string) ([]byte, error) {
		if name == "custom.md" {
			t.Fatal("read the embedded file even though the user has a copy")
		}
		if name == "racing.md" {
			writeUserFile(name)
		}
		return []byte("complete default"), nil
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"custom.md": "user content", "default.md": "complete default", "racing.md": "user content"}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != len(want) {
		t.Fatalf("unexpected files after publication: %v %v", entries, err)
	}
	for name, expected := range want {
		body, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil || string(body) != expected {
			t.Fatalf("%s: %q, %v", name, body, err)
		}
	}
}
