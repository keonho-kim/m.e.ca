// Purpose: Create an isolated single-session database for persistence behavior tests.
// Pattern: Test fixture.
// Usage: SQLite tests own the returned Store and close it after use.
// Related: internal/storage/sqlite/catalog.go, internal/storage/sqlite/store_test.go.
package sqlite

import (
	"context"
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) (*Store, error) {
	t.Helper()
	sessions, err := OpenSessions(filepath.Join(t.TempDir(), "sessions"))
	if err != nil {
		return nil, err
	}
	project := t.TempDir()
	return sessions.Create(context.Background(), project, project)
}
