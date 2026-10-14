//go:build linux

// Purpose: Verify embedded helper extraction refuses corruption and symlink replacement.
// Pattern: Private executable integrity tests.
// Usage: Prepare Linux assets, then go test ./internal/tools/local/process/sandbox/bundle.
// Related: internal/tools/local/process/sandbox/bundle/extract_linux.go, scripts/build-bwrap.sh.
package bundle

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEmbeddedHelperIntegrity(t *testing.T) {
	private := t.TempDir()
	path, err := Executable(private)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0700 {
		t.Fatalf("executable permissions: %v %v", info, err)
	}
	again, err := Executable(private)
	if err != nil || again != path {
		t.Fatalf("helper was not reused: %s %v", again, err)
	}
	if err := os.WriteFile(path, []byte("corrupt"), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := Executable(private); err == nil {
		t.Fatal("corrupt helper accepted")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(filepath.Dir(path), "COPYING"), path); err != nil {
		t.Fatal(err)
	}
	if _, err := Executable(private); err == nil {
		t.Fatal("symlink helper accepted")
	}
}
