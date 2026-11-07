// Purpose: Verify bounded catalog scans preserve ordering and release large scans on cancellation or error.
// Pattern: Persistence lifecycle contract tests.
// Usage: Exercise both sides of the reader admission threshold with the race detector.
// Related: internal/storage/sqlite/conversation_scan.go, internal/storage/sqlite/conversation_reader.go.
package sqlite

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestConversationParallelScanOrderingAndFailure(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	catalog, err := OpenSessions(filepath.Join(root, "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	for range 36 {
		store, err := catalog.Create(ctx, root, root)
		if err != nil {
			t.Fatal(err)
		}
		paths = append(paths, store.path)
		if err := store.Close(); err != nil {
			t.Fatal(err)
		}
	}
	first, err := catalog.Conversations(ctx, "")
	if err != nil || len(first) != len(paths) {
		t.Fatalf("scan: %d %v", len(first), err)
	}
	for range 3 {
		again, err := catalog.Conversations(ctx, "")
		if err != nil || len(again) != len(first) {
			t.Fatalf("scan: %v", err)
		}
		for i := range first {
			if first[i] != again[i] {
				t.Fatal("parallel scan changed metadata order")
			}
		}
	}
	for range 3 {
		ctx, cancel := context.WithCancel(ctx)
		err := catalog.scanConversations(ctx, "", func(conversationResult) { cancel() })
		cancel()
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelled scan: %v", err)
		}
	}
	if err := os.WriteFile(paths[0], []byte("not a database"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.Conversations(ctx, ""); err == nil {
		t.Fatal("corrupt database accepted")
	}
	if err := os.Remove(paths[0]); err != nil {
		t.Fatal(err)
	}
	if values, err := catalog.Conversations(ctx, ""); err != nil || len(values) != len(paths)-1 {
		t.Fatalf("failed scan retained resources: %d %v", len(values), err)
	}
}

func TestConversationReaderRechecksEachAttachedDatabase(t *testing.T) {
	ctx, root := context.Background(), t.TempDir()
	catalog, err := OpenSessions(filepath.Join(root, "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	var files []conversationFile
	for range 2 {
		store, err := catalog.Create(ctx, root, root)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, conversationFile{store.path, store.Session().ID, filepath.Base(filepath.Dir(store.path))})
		if err := store.Close(); err != nil {
			t.Fatal(err)
		}
	}
	reader, err := openConversationReader(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := reader.Close(); err != nil {
			t.Error(err)
		}
	})
	for _, file := range files {
		result, err := reader.Read(ctx, file)
		if err != nil || result.conversation.ID != file.id {
			t.Fatalf("reader reused metadata across databases: %+v %v", result, err)
		}
	}
	db, err := connect(ctx, files[1].path, false)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(ctx, "PRAGMA user_version=1")
	closeErr := db.Close()
	if err != nil || closeErr != nil {
		t.Fatalf("fixture: %v %v", err, closeErr)
	}
	if result, err := reader.Read(ctx, files[1]); err != nil || result.conversation.ID != files[1].id || result.conversation.Unavailable == "" {
		t.Fatal("schema version was not rechecked for the current file", result, err)
	}
}
