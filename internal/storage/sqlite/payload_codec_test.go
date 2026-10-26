// Purpose: Verify persisted Zstandard content is restored after reopening a session database.
// Pattern: Persistence round-trip contract test.
// Usage: go test ./internal/storage/sqlite -run TestPayloadZstandard.
// Related: internal/storage/sqlite/payload.go, internal/storage/content/compression.go.
package sqlite

import (
	"context"
	"strings"
	"testing"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

func TestPayloadZstandardStorageAndReopen(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	text := strings.Repeat("payload line\n", 10000)
	if err := s.RecordEvent(context.Background(), s.Session().ID, runrecord.Event{Kind: "thinking", Thinking: text}); err != nil {
		t.Fatal(err)
	}
	var codec int
	if err := s.db.QueryRow("SELECT codec FROM payloads").Scan(&codec); err != nil || codec != 2 {
		t.Fatal("expected Zstandard payload", codec, err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	db, err := connect(context.Background(), s.path, false)
	if err != nil {
		t.Fatal(err)
	}
	reopened := &Store{db: db, path: s.path, session: s.session}
	t.Cleanup(func() { reopened.Close() })
	page, _, err := reopened.Timeline(context.Background(), s.Session().ID, 0, 10)
	if err != nil || len(page) != 1 || page[0].Event.Thinking != text {
		t.Fatal("stored payload not restored after reopening", err)
	}
}
