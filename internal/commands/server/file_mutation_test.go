// Purpose: Verify file effects and reference remapping finish together even after the initiating client cancels.
// Pattern: Real filesystem and SQLite application contract tests.
// Usage: go test -race ./internal/commands/server -run TestEntryMutation.
// Related: internal/commands/server/file_mutation.go, internal/storage/sqlite/file_selection_changes.go.
package server

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/keonho-kim/meca/internal/fileview"
	"github.com/keonho-kim/meca/internal/policy"
	"github.com/keonho-kim/meca/internal/storage/sqlite"
	"github.com/keonho-kim/meca/internal/tools/local"
)

func TestEntryMutationReconcilesReferencesAfterClientCancellation(t *testing.T) {
	project := t.TempDir()
	if err := os.Mkdir(filepath.Join(project, "old"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "old", "one.txt"), []byte("one"), 0600); err != nil {
		t.Fatal(err)
	}
	sessions, err := sqlite.OpenSessions(filepath.Join(t.TempDir(), "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	store, err := sessions.Create(t.Context(), project, project)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	files, err := local.NewExecutor(project, project, filepath.Join(project, ".meca"), func(text string) string { return text }, local.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer files.Close()
	s := detailFixture(store, nil)
	defer s.Close()
	s.conversation.Runtime.Session = store.Session()
	s.options.Files = files.Files
	id := store.Session().ID
	for _, path := range []string{"old/", "old/one.txt", "other.txt"} {
		if err := store.SelectFile(t.Context(), id, path, true); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s.options.MutateEntry = func(ctx context.Context, settings policy.Settings, change fileview.EntryMutation) (string, error) {
		path, err := files.Files.MutateEntry(ctx, settings, change, func(string) error { return nil })
		cancel()
		return path, err
	}
	page, err := files.Files.Directory(ctx, policy.Settings{Scope: policy.Directory}, ".", 0, 100, "")
	if err != nil {
		t.Fatal(err)
	}
	result, err := s.MutateEntry(ctx, id, fileview.EntryMutation{Path: "old", Name: "new", Revision: page.Revision})
	if err != nil || result.Failure != nil || !result.ReferencesReady {
		t.Fatalf("client cancellation interrupted reconciliation after rename: %+v %v", result, err)
	}
	want := []string{"new/", "new/one.txt", "other.txt"}
	if !reflect.DeepEqual(result.Paths, want) {
		t.Fatalf("authoritative references: %v, want %v", result.Paths, want)
	}
	actual, err := store.FileSelections(t.Context(), id)
	if err != nil || !reflect.DeepEqual(actual, want) {
		t.Fatalf("reference transaction did not persist: %v %v", actual, err)
	}
}
