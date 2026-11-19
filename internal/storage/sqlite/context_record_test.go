// Purpose: Verify direct local content projection preserves field selection, byte accounting and integrity.
// Pattern: SQLite payload and message projection contract test.
// Usage: go test ./internal/storage/sqlite -run TestLocalContentProjection.
// Related: internal/storage/sqlite/context_record.go, internal/storage/sqlite/payload.go.
package sqlite

import (
	"strings"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
)

func TestLocalContentProjectionMatchesRestoredSizeAndChecksIntegrity(t *testing.T) {
	ctx := t.Context()
	store, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	text := strings.Repeat("한글 <>& \"quoted\" \\ line\n", 2048)
	if err := store.AppendMessage(ctx, store.Session().ID, modelapi.Message{Role: "assistant", Content: text, Reasoning: "internal note"}); err != nil {
		t.Fatal(err)
	}
	page, err := store.decisionPage(ctx, 0, 0, false)
	if err != nil || len(page) != 1 || len(page[0].refs) <= 2 {
		t.Fatalf("expected one stored message with a payload reference: %v", err)
	}
	record := page[0]
	for _, fields := range []map[string]bool{nil, {"role": true, "content": true}} {
		restored, err := store.unpackRecord(ctx, record, fields)
		if err != nil {
			t.Fatal(err)
		}
		wantReasoning := "internal note"
		if fields != nil {
			wantReasoning = ""
		}
		got, size, err := store.contextRecord(ctx, record, fields, nil)
		if err != nil || got.Content != text || got.Reasoning != wantReasoning || got.Role != "assistant" || size != len(restored) {
			t.Fatalf("local projection size=%d want=%d reason=%q error=%v", size, len(restored), got.Reasoning, err)
		}
	}
	for _, body := range []string{`{"role":"assistant"}`, `{"role":"assistant","content":"conflict"}`} {
		if _, _, err := store.contextRecord(ctx, storedRecord{body: []byte(body), refs: record.refs}, nil, nil); err == nil {
			t.Fatal("accepted an invalid local content placeholder")
		}
	}
	if _, err := store.db.ExecContext(ctx, "UPDATE payloads SET digest=zeroblob(length(digest))"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.contextRecord(ctx, record, nil, nil); err == nil {
		t.Fatal("accepted local content with a mismatched digest")
	}
}
