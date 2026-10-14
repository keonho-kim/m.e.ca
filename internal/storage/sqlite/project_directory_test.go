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
	// A corrupt database in another project must not be read during local restore.
	if err := os.WriteFile(paths[1], []byte("invalid database"), 0600); err != nil {
		t.Fatal(err)
	}
	restored, err := catalog.Restore(ctx, projects[0], projects[0], true)
	if err != nil {
		t.Fatal(err)
	}
	if restored.path != paths[0] {
		t.Fatal("restored another project")
	}
	if err := restored.Close(); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(projects[0], alias); err != nil {
		t.Fatal(err)
	}
	linked, err := catalog.Restore(ctx, alias, alias, true)
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

func TestProjectDirectoryRejectsMisplacedAndFlatFiles(t *testing.T) {
	for _, flat := range []bool{false, true} {
		t.Run(map[bool]string{false: "wrong-project", true: "flat"}[flat], func(t *testing.T) {
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
			target := catalog.directory
			project := t.TempDir()
			if !flat {
				target, _, err = catalog.projectDirectory(project, true)
				if err != nil {
					t.Fatal(err)
				}
			}
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(target, id+".db"), content, 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := catalog.Conversations(ctx, 0, 100); err == nil {
				t.Fatal("misplaced database was listed")
			}
			if !flat {
				if store, err := catalog.Open(ctx, project, id); err == nil {
					if err := store.Close(); err != nil {
						t.Error(err)
					}
					t.Fatal("opened another project's database")
				}
			}
			if _, err := os.Stat(filepath.Join(target, id+".db")); err != nil {
				t.Fatal("unsupported file was removed")
			}
		})
	}
}
