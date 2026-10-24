//go:build unix

// Purpose: Verify capture reclamation skips active owners and removes abandoned output directories.
// Pattern: Filesystem lease lifecycle test.
// Usage: go test ./internal/tools/local -run TestArtifactLease.
// Related: internal/tools/local/artifacts_unix.go, internal/tools/local/executor.go.
package local

import (
	"os"
	"testing"
)

func TestArtifactLeaseReclaimsOnlyInactiveDirectories(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	first, lease, err := newArtifacts()
	if err != nil {
		t.Fatal(err)
	}
	second, active, err := newArtifacts()
	if err != nil {
		lease.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { active.Close() })
	if _, err := os.Stat(first); err != nil {
		t.Fatal("active capture reclaimed", err)
	}
	if err := lease.Close(); err != nil {
		t.Fatal(err)
	}
	_, third, err := newArtifacts()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { third.Close() })
	if _, err := os.Stat(first); !os.IsNotExist(err) {
		t.Fatal("abandoned capture retained", err)
	}
	if _, err := os.Stat(second); err != nil {
		t.Fatal("another active capture reclaimed", err)
	}
}
