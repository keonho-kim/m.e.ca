// Purpose: Verify code result envelopes share originals and restore exact durable JSON.
// Pattern: SQLite storage contract tests with deterministic incompressible content.
// Usage: go test ./internal/storage/sqlite -run TestCodeResult.
// Related: internal/storage/sqlite/prepared_payload.go, internal/storage/sqlite/payload.go.
package sqlite

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/rand"
	"strconv"
	"strings"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/storage/content"
)

func digestBytes(body []byte) []byte {
	digest := sha256.Sum256(body)
	return digest[:]
}

func TestCodeResultRechecksOriginalBeforeCommit(t *testing.T) {
	store, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	original, _ := json.Marshal(map[string]string{"text": strings.Repeat("shared", 10000)})
	call := toolapi.Call{ID: "read", Name: "execute_command", Arguments: json.RawMessage(`{}`)}
	event := runrecord.Event{Kind: "tool_succeeded", CallID: call.ID, Call: &call, Output: original}
	if err := store.RecordEvent(t.Context(), store.Session().ID, event); err != nil {
		t.Fatal(err)
	}
	message := modelapi.Message{Role: "tool", ToolName: "codemode", ToolCallID: "code", Content: `{"status":"completed","result":` + string(original) + `,"tool_calls":1}`}
	prepared, err := store.prepareJournal(t.Context(), []runrecord.Entry{{Message: &message}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(t.Context(), "DELETE FROM payloads WHERE digest=?", digestBytes(original)); err != nil {
		t.Fatal(err)
	}
	tx, err := store.db.BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if err := prepared[0].write(t.Context(), store, tx, store.Session().ID, "now", &journalReceipt{}); err == nil {
		t.Fatal("code result committed a missing original reference")
	}
}

func TestCodeResultSharesNativeOriginal(t *testing.T) {
	for _, repeated := range []bool{false, true} {
		t.Run(strconv.FormatBool(repeated), func(t *testing.T) {
			store, err := newTestStore(t)
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			input := make([]byte, 64<<10)
			rand.New(rand.NewSource(1)).Read(input)
			original, _ := json.Marshal(map[string]string{"body": base64.StdEncoding.EncodeToString(input)})
			call := toolapi.Call{ID: "inner", Name: "execute_command", Arguments: json.RawMessage(`{}`)}
			event := runrecord.Event{Kind: "tool_succeeded", CallID: call.ID, Call: &call, Output: original}
			if err := store.RecordEvent(t.Context(), store.Session().ID, event); err != nil {
				t.Fatal(err)
			}
			value := original
			if repeated {
				value, _ = json.Marshal([]json.RawMessage{original, original})
			}
			body, _ := json.Marshal(struct {
				Status string          `json:"status"`
				Result json.RawMessage `json:"result"`
				Calls  int             `json:"tool_calls"`
			}{"completed", value, 1})
			parent := toolapi.Call{ID: "code", Name: "codemode", Arguments: json.RawMessage(`{}`)}
			message := modelapi.Message{Role: "tool", ToolName: parent.Name, ToolCallID: parent.ID, Content: string(body)}
			event = runrecord.Event{Kind: "tool_succeeded", CallID: parent.ID, Call: &parent, Output: body}
			if err := store.Commit(t.Context(), store.Session().ID, []runrecord.Entry{{Message: &message}, {Event: &event}}); err != nil {
				t.Fatal(err)
			}
			var stored int
			if err := store.db.QueryRowContext(t.Context(), "SELECT SUM(length(body)) FROM payloads").Scan(&stored); err != nil {
				t.Fatal(err)
			}
			if stored > len(original)+2048 {
				t.Fatalf("original was duplicated: original=%d stored=%d", len(original), stored)
			}
			t.Logf("original=%d result=%d stored=%d", len(original), len(body), stored)
			messages, err := store.ContextMessages(t.Context(), store.Session().ID, 0, 8<<20, false, nil)
			if err != nil || len(messages) != 1 || messages[0].Content != string(body) {
				t.Fatalf("restored message mismatch: %v", err)
			}
			var id int64
			if err := store.db.QueryRowContext(t.Context(), "SELECT id FROM payloads WHERE digest=?", digestBytes(body)).Scan(&id); err != nil {
				t.Fatal(err)
			}
			raw, err := store.payload(t.Context(), strconv.FormatInt(id, 10))
			if err != nil || !bytes.Equal(raw, body) {
				t.Fatalf("restored original mismatch: %v", err)
			}
			prefix, clipped, err := store.readTextPrefix(t.Context(), content.StringReference(strconv.FormatInt(id, 10)), newReadBudget(8<<20, nil), 64)
			if err != nil || !clipped || prefix != string(body[:64]) {
				t.Fatalf("prefix=%q clipped=%v err=%v", prefix, clipped, err)
			}
			if _, err := store.db.ExecContext(t.Context(), "UPDATE payloads SET body=? WHERE digest=?", []byte(`{}`), digestBytes(original)); err != nil {
				t.Fatal(err)
			}
			if _, err := store.payload(t.Context(), strconv.FormatInt(id, 10)); err == nil {
				t.Fatal("corrupted shared original was accepted")
			}
		})
	}
}
