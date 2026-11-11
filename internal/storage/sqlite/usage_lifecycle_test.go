// Purpose: Keep reference-only handles from allocating or evicting verified token projections.
// Pattern: Repository resource-lifecycle regression.
// Usage: go test -race ./internal/storage/sqlite -run TestReferenceHandles.
// Related: internal/storage/sqlite/catalog.go, internal/storage/sqlite/usage.go, internal/storage/sqlite/project_references.go.
package sqlite

import (
	"context"
	"path/filepath"
	"testing"
)

func TestReferenceHandlesDoNotPopulateTokenUsageCache(t *testing.T) {
	ctx, project := t.Context(), t.TempDir()
	sessions, err := OpenSessions(filepath.Join(t.TempDir(), "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	first, err := sessions.Create(ctx, project, project)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	second, err := sessions.Create(ctx, project, project)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	references := ProjectReferences{Sessions: sessions, Project: project}
	for _, store := range []*Store{first, second} {
		if _, err := references.List(ctx, store.Session().ID); err != nil {
			t.Fatal(err)
		}
	}
	if len(sessions.usage.entries) != 0 {
		t.Fatal("reference-only handles allocated unused token aggregates")
	}
	if _, err := first.TokenUsage(ctx, first.Session().ID); err != nil {
		t.Fatal(err)
	}
	cached := first.usage
	if cached == nil || len(sessions.usage.entries) != 1 {
		t.Fatal("usage query did not bind its shared projection")
	}
	if err := references.Rewrite(ctx, func(_ context.Context, path string) (string, error) { return path, nil }, func(string, []string) {}); err != nil {
		t.Fatal(err)
	}
	if len(sessions.usage.entries) != 1 || sessions.usage.entries[first.path] != cached {
		t.Fatal("reference scan polluted the verified token cache")
	}
}
