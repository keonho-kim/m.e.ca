// Purpose: Verify source captures and execution snapshots share exact file versions.
// Pattern: Filesystem contract tests.
// Usage: File preparation and mutation guards rely on these capture invariants.
// Related: internal/tools/local/version.go, internal/tools/local/candidate.go.
package local

import (
	"bytes"
	"crypto/sha256"
	"os"
	"path/filepath"
	"testing"

	"github.com/keonho-kim/meca/internal/policy"
)

func TestCapturedSourceMatchesExecutionVersion(t *testing.T) {
	e, root := testExecutor(t)
	for _, body := range [][]byte{nil, []byte("한글\r\nlast"), bytes.Repeat([]byte("source\n"), 50000)} {
		if err := os.WriteFile(filepath.Join(root, "source"), body, 0600); err != nil {
			t.Fatal(err)
		}
		path, err := e.resolve(policy.System, "source")
		if err != nil {
			t.Fatal(err)
		}
		captured, content, err := capture(path, true)
		if err != nil || !bytes.Equal(content, body) || captured.digest != sha256.Sum256(body) {
			t.Fatalf("source capture: %d bytes, %v", len(content), err)
		}
		current, err := snapshot(path)
		if err != nil || !sameVersion(captured, current) {
			t.Fatalf("execution version differs: %v", err)
		}
		if err := os.WriteFile(filepath.Join(root, "source"), append(bytes.Clone(body), 'x'), 0600); err != nil {
			t.Fatal(err)
		}
		changed, err := snapshot(path)
		if err != nil || sameVersion(captured, changed) {
			t.Fatalf("changed source accepted: %v", err)
		}
	}
}

func TestCaptureMissingDirectoryAndOversizedSource(t *testing.T) {
	e, root := testExecutor(t)
	for _, name := range []string{"missing", "dir", "large"} {
		path, err := e.resolve(policy.System, name)
		if err != nil {
			t.Fatal(err)
		}
		if name == "dir" {
			if err := os.Mkdir(filepath.Join(root, name), 0700); err != nil {
				t.Fatal(err)
			}
		}
		if name == "large" {
			if err := os.WriteFile(filepath.Join(root, name), nil, 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.Truncate(filepath.Join(root, name), 16*1024*1024+1); err != nil {
				t.Fatal(err)
			}
		}
		value, body, err := capture(path, true)
		if name == "large" {
			if err == nil {
				t.Fatal("oversized source accepted")
			}
			continue
		}
		if err != nil || body != nil || value.exists != (name == "dir") {
			t.Fatalf("%s: %+v %v", name, value, err)
		}
	}
}
