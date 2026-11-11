// Purpose: Verify workspace entry writes reject stale lists, collisions and protected descendants.
// Pattern: Native filesystem integration tests.
// Usage: go test ./internal/tools/local -run TestWorkspaceEntryMutations.
// Related: internal/fileview/entry_mutation.go, internal/tools/local/filesystem/paths.go.
package local

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/keonho-kim/meca/internal/fileview"
	"github.com/keonho-kim/meca/internal/policy"
)

func TestWorkspaceEntryMutations(t *testing.T) {
	root := t.TempDir()
	executor, err := NewExecutor(root, root, filepath.Join(root, ".meca"), func(s string) string { return s }, Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer executor.Close()
	settings := policy.Settings{Scope: policy.Directory}
	ctx := t.Context()
	if err := os.MkdirAll(filepath.Join(root, ".config", "gh"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "folder"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "source.txt"), []byte("source"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "folder", "renamed.txt"), []byte("destination"), 0600); err != nil {
		t.Fatal(err)
	}
	revision := func() string {
		t.Helper()
		page, err := executor.Files.Directory(ctx, settings, ".", 0, 100, "")
		if err != nil {
			t.Fatal(err)
		}
		return page.Revision
	}
	mutate := func(change fileview.EntryMutation) error {
		t.Helper()
		_, err := executor.Files.MutateEntry(ctx, settings, change, executor.boundary.CheckEntryMutation)
		return err
	}
	original := revision()
	if err := mutate(fileview.EntryMutation{Path: "source.txt", Name: "SOURCE.txt", Revision: original}); err != nil {
		t.Fatalf("case-only rename failed: %v", err)
	}
	if err := mutate(fileview.EntryMutation{Path: "SOURCE.txt", Name: "source.txt", Revision: revision()}); err != nil {
		t.Fatal(err)
	}
	original = revision()
	if err := mutate(fileview.EntryMutation{Path: "source.txt", Name: "renamed.txt", Revision: original}); err != nil {
		t.Fatal(err)
	}
	if err := mutate(fileview.EntryMutation{Path: "renamed.txt", Remove: true, Revision: original}); err == nil {
		t.Fatal("stale revision removed a renamed entry")
	}
	if err := mutate(fileview.EntryMutation{Path: "renamed.txt", Folder: "folder", Revision: revision()}); err == nil {
		t.Fatal("move replaced an existing destination")
	}
	if body, err := os.ReadFile(filepath.Join(root, "folder", "renamed.txt")); err != nil || string(body) != "destination" {
		t.Fatal("existing destination changed", string(body), err)
	}
	if err := os.Remove(filepath.Join(root, "folder", "renamed.txt")); err != nil {
		t.Fatal(err)
	}
	if err := mutate(fileview.EntryMutation{Path: "renamed.txt", Folder: "folder", Revision: revision()}); err != nil {
		t.Fatal(err)
	}
	if body, err := os.ReadFile(filepath.Join(root, "folder", "renamed.txt")); err != nil || string(body) != "source" {
		t.Fatal("file did not move", string(body), err)
	}
	if err := mutate(fileview.EntryMutation{Path: ".config", Remove: true, Revision: revision()}); err == nil {
		t.Fatal("mutation removed a directory containing protected state")
	}
	if err := mutate(fileview.EntryMutation{Path: "folder", Remove: true, Revision: revision()}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "folder")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("folder remains after removal", err)
	}
}
