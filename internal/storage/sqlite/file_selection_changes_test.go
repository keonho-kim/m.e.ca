// Purpose: Verify batch remapping is atomic, deduplicated and based on the original selections.
// Pattern: Repository transaction contract test.
// Usage: go test ./internal/storage/sqlite -run TestFileSelectionChanges.
// Related: internal/storage/sqlite/file_selection_changes.go.
package sqlite

import (
	"context"
	"reflect"
	"testing"
)

func TestFileSelectionChangesReplaceOnceAndKeepFailedBatchUnchanged(t *testing.T) {
	store, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx, id := t.Context(), store.Session().ID
	for _, path := range []string{"a", "b", "keep", "remove/"} {
		if err := store.SelectFile(ctx, id, path, true); err != nil {
			t.Fatal(err)
		}
	}
	changes := map[string]string{"a": "b", "b": "c", "remove/": "", "missing": "unexpected"}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := store.ChangeFileSelections(canceled, id, changes); err == nil {
		t.Fatal("canceled batch succeeded")
	}
	before, err := store.FileSelections(ctx, id)
	if err != nil || !reflect.DeepEqual(before, []string{"a", "b", "keep", "remove/"}) {
		t.Fatalf("failed batch changed selections: %v %v", before, err)
	}
	after, err := store.ChangeFileSelections(ctx, id, changes)
	if err != nil || !reflect.DeepEqual(after, []string{"b", "c", "keep"}) {
		t.Fatalf("replacement cascaded or lost selected destinations: %v %v", after, err)
	}
	if _, err := store.ChangeFileSelections(ctx, id, map[string]string{"b": "c"}); err != nil {
		t.Fatal(err)
	}
	after, err = store.FileSelections(ctx, id)
	if err != nil || !reflect.DeepEqual(after, []string{"c", "keep"}) {
		t.Fatalf("duplicate destination: %v %v", after, err)
	}
}
