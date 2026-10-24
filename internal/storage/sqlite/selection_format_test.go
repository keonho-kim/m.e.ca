// Purpose: Reject unknown persisted selection fields while preserving the stored bytes.
// Pattern: Storage boundary contract test.
// Usage: go test ./internal/storage/sqlite -run TestSelectionRejectsUnsupportedFields.
// Related: internal/storage/sqlite/selection.go, internal/provider/selection/validate.go.
package sqlite

import (
	"context"
	"encoding/json"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"testing"
)

func TestSelectionRejectsUnsupportedFields(t *testing.T) {
	store, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			t.Error(err)
		}
	}()
	ctx, session := context.Background(), store.Session()
	conn, err := connection.New(connection.VLLM, "")
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(connection.Selection{Connection: conn, Model: "alias"})
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatal(err)
	}
	fields["unexpected"] = json.RawMessage(`true`)
	body, err := json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, "UPDATE sessions SET selection=? WHERE id=?", body, session.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Selection(ctx, session.ID); err == nil {
		t.Fatal("unknown stored field accepted")
	}
	var retained []byte
	if err := store.db.QueryRowContext(ctx, "SELECT selection FROM sessions WHERE id=?", session.ID).Scan(&retained); err != nil {
		t.Fatal(err)
	}
	if string(retained) != string(body) {
		t.Fatal("unsupported selection was modified")
	}
}
