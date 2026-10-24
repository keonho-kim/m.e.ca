// Purpose: Verify per-call token records aggregate once across persisted conversation recovery.
// Pattern: Repository contract test.
// Usage: go test ./internal/storage/sqlite -run TestTokenUsage.
// Related: internal/storage/sqlite/usage.go, internal/storage/sqlite/events.go.
package sqlite

import (
	"context"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/run"
	"testing"
)

func TestTokenUsagePersistsAndRejectsInvalidCounts(t *testing.T) {
	store, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Error(err)
		}
	})
	ctx := context.Background()
	session := store.Session()
	for _, usage := range []modelapi.TokenUsage{{Input: 12, Output: 3}, {Input: 8, Output: 5}} {
		if err := store.RecordEvent(ctx, session.ID, run.Event{Kind: "usage", Usage: &usage}); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.RecordEvent(ctx, session.ID, run.Event{Kind: "usage", Usage: &modelapi.TokenUsage{Input: -1}}); err == nil {
		t.Fatal("negative token count persisted")
	}
	total, err := store.TokenUsage(ctx, session.ID)
	if err != nil || total.Input != 20 || total.Output != 8 {
		t.Fatalf("total=%+v err=%v", total, err)
	}
	unsupported := `{"kind":"usage","usage":{"input":100,"output":50,"estimated":true}}`
	if _, err := store.db.ExecContext(ctx, "INSERT INTO tool_events(session_id,kind,body,raw_bytes,created_at) VALUES(?,'usage',?,?,?)", session.ID, unsupported, len(unsupported), "2026-10-13T00:00:00Z"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.TokenUsage(ctx, session.ID); err == nil {
		t.Fatal("estimated record accepted as actual usage")
	}
	var retained string
	if err := store.db.QueryRowContext(ctx, "SELECT body FROM tool_events WHERE session_id=? ORDER BY sequence DESC LIMIT 1", session.ID).Scan(&retained); err != nil || retained != unsupported {
		t.Fatalf("unsupported record modified: %q %v", retained, err)
	}
}
