// Purpose: Verify cancellable tree removal stays inside pinned directories and preserves external symlink targets.
// Pattern: Filesystem traversal regression tests.
// Usage: go test ./internal/fileview -run TestRemoveEntry.
// Related: internal/fileview/entry_remove.go.
package fileview

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

type removalContext struct {
	context.Context
	remaining int
	cancel    context.CancelFunc
}

func (c *removalContext) Err() error {
	c.remaining--
	if c.remaining == 0 {
		c.cancel()
	}
	return c.Context.Err()
}

func TestRemoveEntryCancellationAndSymlinks(t *testing.T) {
	directory := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "keep.txt"), []byte("retained"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(directory, "tree"), 0700); err != nil {
		t.Fatal(err)
	}
	for i := range 200 {
		if err := os.WriteFile(filepath.Join(directory, "tree", fmt.Sprint(i)), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(outside, filepath.Join(directory, "tree", "outside")); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	// Cancel during traversal, after some effects have completed.
	err = removeEntry(&removalContext{Context: ctx, remaining: 12, cancel: cancel}, root, "tree")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("tree deletion ignored cancellation: %v", err)
	}
	remaining, err := os.ReadDir(filepath.Join(directory, "tree"))
	if err != nil || len(remaining) == 0 || len(remaining) >= 201 {
		t.Fatalf("expected explicit partial deletion: %d %v", len(remaining), err)
	}
	if err := removeEntry(t.Context(), root, "tree"); err != nil {
		t.Fatal(err)
	}
	if _, err := root.Stat("tree"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("tree remains after completion: %v", err)
	}
	if body, err := os.ReadFile(filepath.Join(outside, "keep.txt")); err != nil || string(body) != "retained" {
		t.Fatalf("symlink target was changed: %q %v", body, err)
	}
}
