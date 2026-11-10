// Purpose: Verify revision-pinned directory pages retain policy checks and independent output slices.
// Pattern: Guarded filesystem integration tests.
// Usage: go test -race ./internal/fileview -run TestDirectorySnapshot.
// Related: internal/fileview/directory_snapshot.go.
package fileview

import (
	"errors"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/policy"
	"os"
	"path/filepath"
	"testing"
)

func TestDirectorySnapshotRevisionAndCurrentVisibility(t *testing.T) {
	base := t.TempDir()
	for _, name := range []string{"one", "two"} {
		if err := os.WriteFile(filepath.Join(base, name), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	root, err := os.OpenRoot(base)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	denied := false
	reader := New(base, func(_ policy.Scope, path string) (Target, error) {
		if denied && filepath.Base(path) == "one" {
			return Target{}, os.ErrPermission
		}
		if path == "." {
			path = base
		}
		relative, err := filepath.Rel(base, path)
		return Target{Absolute: path, Relative: relative, Root: root}, err
	}, func(s string) string { return s })
	settings := policy.Settings{Scope: policy.System}
	first, err := reader.Directory(t.Context(), settings, ".", 0, 1, "")
	if err != nil || first.Revision == "" || !first.More {
		t.Fatal(first, err)
	}
	first.Entries[0].Name = "caller mutation"
	second, err := reader.Directory(t.Context(), settings, ".", 0, 2, first.Revision)
	if err != nil || second.Entries[0].Name != "one" {
		t.Fatal(second, err)
	}
	denied = true
	filtered, err := reader.Directory(t.Context(), settings, ".", 0, 2, first.Revision)
	if err != nil || len(filtered.Entries) != 1 || filtered.Entries[0].Name != "two" {
		t.Fatal(filtered, err)
	}
	previous, err := os.Stat(base)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "three"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(base, previous.ModTime(), previous.ModTime()); err != nil {
		t.Fatal(err)
	}
	_, err = reader.Directory(t.Context(), settings, ".", 0, 2, first.Revision)
	var typed *fault.Error
	if !errors.As(err, &typed) || string(typed.ErrorCode()) != "conflict" {
		t.Fatal("stale directory revision accepted", err)
	}
}
