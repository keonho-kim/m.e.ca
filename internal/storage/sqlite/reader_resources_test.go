// Purpose: Verify repeated catalogue reads and mid-scan cancellation release reader file descriptors.
// Pattern: Persistence resource lifecycle tests.
// Usage: go test ./internal/storage/sqlite -run TestConversationReaderResources.
// Related: internal/storage/sqlite/conversation_scan.go, internal/storage/sqlite/conversations.go.
package sqlite

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestConversationReaderResources(t *testing.T) {
	root := t.TempDir()
	catalog, err := OpenSessions(filepath.Join(root, "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	for range 8 {
		store, err := catalog.Create(context.Background(), root, root)
		if err != nil {
			t.Fatal(err)
		}
		if err := store.Close(); err != nil {
			t.Fatal(err)
		}
	}
	fdPath := "/dev/fd"
	if runtime.GOOS == "linux" {
		fdPath = "/proc/self/fd"
	}
	before, err := os.ReadDir(fdPath)
	if err != nil {
		t.Skipf("descriptor inventory unavailable: %v", err)
	}
	for range 40 {
		ctx, cancel := context.WithCancel(context.Background())
		err := catalog.scanConversations(ctx, "", func(conversationResult) { cancel() })
		cancel()
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("mid-scan cancellation: %v", err)
		}
		if rows, err := catalog.Conversations(context.Background(), ""); err != nil || len(rows) != 8 {
			t.Fatalf("subsequent scan: %d %v", len(rows), err)
		}
	}
	after, err := os.ReadDir(fdPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) > len(before)+2 {
		t.Fatalf("reader descriptors grew from %d to %d", len(before), len(after))
	}
}
