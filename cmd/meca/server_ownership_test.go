//go:build !minimum

// Purpose: Reject a second server composition before opening its ledger or starting a listener.
// Pattern: Composition-root ownership regression test.
// Usage: go test ./cmd/meca -run TestServerRejectsExistingOwner.
// Related: cmd/meca/server.go, internal/storage/sqlite/locks_unix.go.
package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/commands/startup"
	"github.com/keonho-kim/meca/internal/config"
	"github.com/keonho-kim/meca/internal/storage/sqlite"
)

func TestServerRejectsExistingOwner(t *testing.T) {
	home, project := t.TempDir(), t.TempDir()
	state := filepath.Join(home, ".meca")
	sessions, err := sqlite.OpenSessions(filepath.Join(state, "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	owner, err := sessions.LeaseServer(project)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	err = runServer(t.Context(), startup.Mode{Project: project}, config.Defaults(), home, state, sessions)
	if err == nil || !strings.Contains(err.Error(), "server ownership") {
		t.Fatal("second server composition was admitted", err)
	}
}
