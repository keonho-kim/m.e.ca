//go:build darwin

// Purpose: Prevent case aliases from bypassing protected paths on case-insensitive macOS volumes.
// Pattern: Native filesystem regression test.
// Usage: go test ./internal/tools/local/filesystem -run TestProtectedCaseAlias.
// Related: internal/tools/local/filesystem/canonical_darwin.go.
package filesystem

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProtectedCaseAlias(t *testing.T) {
	home, project := t.TempDir(), t.TempDir()
	private := filepath.Join(home, ".V")
	if err := os.Mkdir(private, 0700); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(home, ".v")
	if _, err := os.Stat(alias); os.IsNotExist(err) {
		t.Skip("case-sensitive volume")
	}
	b, err := New(project, home, alias)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.ReadTarget(filepath.Join(private, "secret")); err == nil {
		t.Fatal("case alias exposed protected path")
	}
}
