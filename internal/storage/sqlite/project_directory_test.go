// Purpose: Verify project directories isolate discovery and map symlink aliases to the same session group.
// Pattern: Storage layout integration test.
// Usage: go test ./internal/storage/sqlite -run TestProjectDirectory.
// Related: internal/storage/sqlite/directories.go, internal/storage/session/project.go.
package sqlite

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestProjectDirectoryIgnoresUnrelatedRootFiles(t *testing.T) {
	ctx := context.Background()
	catalog, err := OpenSessions(filepath.Join(t.TempDir(), "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	project := t.TempDir()
	store, err := catalog.Create(ctx, project, project)
	if err != nil {
		t.Fatal(err)
	}
	id := store.Session().ID
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	sidecar := filepath.Join(catalog.directory, "session_notes.db")
	if err := os.WriteFile(sidecar, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	conversations, err := catalog.Conversations(ctx, 0, 100)
	if err != nil || len(conversations) != 1 || conversations[0].ID != id {
		t.Fatalf("unrelated file blocked project discovery: %+v %v", conversations, err)
	}
	content, err := os.ReadFile(sidecar)
	if err != nil || string(content) != "keep" {
		t.Fatalf("unrelated file was changed: %q %v", content, err)
	}
}

func TestProjectDirectoryIsolation(t *testing.T) {
	ctx := context.Background()
	catalog, err := OpenSessions(filepath.Join(t.TempDir(), "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	projects := []string{filepath.Join(t.TempDir(), "same"), filepath.Join(t.TempDir(), "same")}
	paths := []string{}
	for _, project := range projects {
		if err := os.Mkdir(project, 0700); err != nil {
			t.Fatal(err)
		}
		store, err := catalog.Create(ctx, project, project)
		if err != nil {
			t.Fatal(err)
		}
		paths = append(paths, store.path)
		if filepath.Dir(store.path) == catalog.directory {
			t.Fatal("database was stored in the flat root")
		}
		if err := store.Close(); err != nil {
			t.Fatal(err)
		}
	}
	if filepath.Dir(paths[0]) == filepath.Dir(paths[1]) {
		t.Fatal("same-named projects share a directory")
	}
	all, err := catalog.Conversations(ctx, 0, 100)
	if err != nil || len(all) != 2 {
		t.Fatalf("all projects: %v %v", all, err)
	}
	// A corrupt database in another project must not be read during local listing.
	if err := os.WriteFile(paths[1], []byte("invalid database"), 0600); err != nil {
		t.Fatal(err)
	}
	local, err := catalog.ProjectConversations(ctx, projects[0], 0, 100)
	if err != nil || len(local) != 1 {
		t.Fatalf("project list: %v %v", local, err)
	}
	restored, err := catalog.Open(ctx, projects[0], local[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if restored.path != paths[0] {
		t.Fatal("opened another project")
	}
	if err := restored.Close(); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(projects[0], alias); err != nil {
		t.Fatal(err)
	}
	linked, err := catalog.Open(ctx, alias, local[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if linked.path != paths[0] {
		t.Fatal("symlink alias created another session group")
	}
	if err := linked.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestProjectDirectoryRejectsMisplacedDatabase(t *testing.T) {
	ctx := context.Background()
	catalog, err := OpenSessions(filepath.Join(t.TempDir(), "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	first, err := catalog.Create(ctx, t.TempDir(), "first")
	if err != nil {
		t.Fatal(err)
	}
	id, path := first.Session().ID, first.path
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	project := t.TempDir()
	target, _, err := catalog.projectDirectory(project, true)
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(target, id+".db")
	if err := os.WriteFile(destination, content, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.Conversations(ctx, 0, 100); err == nil {
		t.Fatal("misplaced database was listed")
	}
	if store, err := catalog.Open(ctx, project, id); err == nil {
		if err := store.Close(); err != nil {
			t.Error(err)
		}
		t.Fatal("opened another project's database")
	}
	if _, err := os.Stat(destination); err != nil {
		t.Fatal("misplaced file was changed")
	}
}
