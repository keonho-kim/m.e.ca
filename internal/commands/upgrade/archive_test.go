// Purpose: Verify extraction never follows archive paths or links and rejects incomplete executables.
// Pattern: Filesystem boundary tests.
// Usage: go test ./internal/commands/upgrade -run TestExtract.
// Related: internal/commands/upgrade/archive.go, internal/commands/upgrade/fixtures_test.go.
package upgrade

import (
	"archive/tar"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractRejectsInvalidExecutableEntries(t *testing.T) {
	for _, tc := range []struct {
		name    string
		entries []archiveEntry
	}{
		{"missing", []archiveEntry{{name: "LICENSE", content: "license"}}},
		{"symlink", []archiveEntry{{name: "meca", kind: tar.TypeSymlink}}},
		{"hard link", []archiveEntry{{name: "meca", kind: tar.TypeLink}}},
		{"duplicate", []archiveEntry{{name: "meca", content: "first"}, {name: "meca", content: "second"}}},
		{"empty", []archiveEntry{{name: "meca"}}},
		{"path traversal", []archiveEntry{{name: "../meca", content: "outside"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := temporaryRoot(t)
			if err := os.WriteFile(filepath.Join(root.Name(), "release.tar.gz"), tarball(t, tc.entries...), 0600); err != nil {
				t.Fatal(err)
			}
			if err := extract(root); err == nil {
				t.Fatal("accepted an invalid executable entry")
			}
		})
	}
}

func TestExtractWritesOnlyExecutable(t *testing.T) {
	root := temporaryRoot(t)
	archive := tarball(t,
		archiveEntry{name: "meca", content: "executable"},
		archiveEntry{name: "LICENSE", content: "license"},
		archiveEntry{name: "nested/file", content: "ignored"},
	)
	if err := os.WriteFile(filepath.Join(root.Name(), "release.tar.gz"), archive, 0600); err != nil {
		t.Fatal(err)
	}
	if err := extract(root); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(root.Name())
	if err != nil || len(entries) != 2 {
		t.Fatalf("unexpected extracted files: %v, %v", entries, err)
	}
}
