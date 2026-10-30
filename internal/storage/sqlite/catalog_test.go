// Purpose: Verify session discovery comes from separate database files and preserves isolation.
// Pattern: Filesystem and SQLite integration tests.
// Usage: go test ./internal/storage/sqlite -run TestSessionFiles.
// Related: internal/storage/sqlite/catalog.go, internal/storage/sqlite/store.go.
package sqlite

import (
	"context"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"os"
	"path/filepath"
	"testing"
)

func TestSessionFilesAreDiscoveredWithoutIndex(t *testing.T) {
	ctx := context.Background()
	directory := filepath.Join(t.TempDir(), "sessions")
	sessions, err := OpenSessions(directory)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	first, err := sessions.Create(ctx, root, root)
	if err != nil {
		t.Fatal(err)
	}
	firstID := first.Session().ID
	group := filepath.Dir(first.path)
	if err := first.AppendMessage(ctx, firstID, modelapi.Message{Role: "user", Content: "first conversation"}); err != nil {
		t.Fatal(err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	second, err := sessions.Create(ctx, root, root)
	if err != nil {
		t.Fatal(err)
	}
	secondID := second.Session().ID
	if firstID == secondID {
		t.Fatal("reused session ID")
	}
	if err := second.Close(); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{firstID, secondID} {
		info, err := os.Stat(filepath.Join(group, id+".db"))
		if err != nil || info.Mode().Perm() != 0600 {
			t.Fatalf("session DB: %v %v", info, err)
		}
	}
	list, err := sessions.Conversations(ctx, "")
	if err != nil || len(list) != 2 {
		t.Fatalf("list=%+v %v", list, err)
	}
	copyDir := filepath.Join(t.TempDir(), "sessions")
	copied, err := OpenSessions(copyDir)
	if err != nil {
		t.Fatal(err)
	}
	copyGroup, _, err := copied.projectDirectory(root, true)
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(group, firstID+".db"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(copyGroup, firstID+".db"), body, 0600); err != nil {
		t.Fatal(err)
	}
	list, err = copied.Conversations(ctx, "")
	if err != nil || len(list) != 1 || list[0].ID != firstID || list[0].Summary != "first conversation" {
		t.Fatalf("file-only resume: %+v %v", list, err)
	}
	resumed, err := copied.Open(ctx, root, firstID)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := resumed.Close(); err != nil {
			t.Error(err)
		}
	}()
	history, err := resumed.ContextMessages(ctx, firstID, 0, 16*1024*1024, true)
	if err != nil || len(history) != 1 {
		t.Fatalf("history=%+v %v", history, err)
	}
}
