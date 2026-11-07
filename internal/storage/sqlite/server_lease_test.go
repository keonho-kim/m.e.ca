// Purpose: Reject a second server owner for the same canonical project while allowing other projects.
// Pattern: Process-shared ownership contract test.
// Usage: go test ./internal/storage/sqlite -run TestServerLease.
// Related: internal/storage/sqlite/locks_unix.go, cmd/meca/server.go.
package sqlite

import (
	"os"
	"path/filepath"
	"testing"
)

func TestServerLeaseCanonicalProjectAndRelease(t *testing.T) {
	sessions, err := OpenSessions(filepath.Join(t.TempDir(), "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	project := t.TempDir()
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(project, alias); err != nil {
		t.Fatal(err)
	}
	first, err := sessions.LeaseServer(project)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if first != nil {
			_ = first.Close()
		}
	}()
	if second, err := sessions.LeaseServer(alias); err == nil {
		_ = second.Close()
		t.Fatal("two server owners acquired the same project")
	}
	other, err := sessions.LeaseServer(t.TempDir())
	if err != nil {
		t.Fatal("independent project was blocked", err)
	}
	if err := other.Close(); err != nil {
		t.Fatal(err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	first = nil
	last, err := sessions.LeaseServer(alias)
	if err != nil {
		t.Fatal("released project stayed locked", err)
	}
	if err := last.Close(); err != nil {
		t.Fatal(err)
	}
}
