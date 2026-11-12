// Purpose: Prevent separate server instances from publishing credentials through the same token file.
// Pattern: Filesystem ownership regression.
// Usage: go test ./internal/serverapi -run TestTokenFileLease.
// Related: internal/serverapi/token_lease_unix.go, cmd/meca/server.go.
package serverapi

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTokenFileLease(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "server.token")
	first, err := LeaseTokenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if first != nil {
			_ = first.Close()
		}
	})
	if second, err := LeaseTokenFile(path); err == nil {
		second.Close()
		t.Fatal("two servers share a token publisher")
	}
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(root, alias); err != nil {
		t.Fatal(err)
	}
	if second, err := LeaseTokenFile(filepath.Join(alias, "server.token")); err == nil {
		second.Close()
		t.Fatal("directory alias bypassed ownership")
	}
	other, err := LeaseTokenFile(filepath.Join(root, "other.token"))
	if err != nil {
		t.Fatal(err)
	}
	other.Close()
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	first = nil
	next, err := LeaseTokenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	next.Close()
}
