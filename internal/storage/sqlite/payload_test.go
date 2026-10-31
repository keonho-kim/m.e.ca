// Purpose: Verify large repeated bodies occupy one compressed payload while public reads retain originals.
// Pattern: SQLite round-trip and physical storage contract tests.
// Usage: go test ./internal/storage/sqlite -run TestPayload.
// Related: internal/storage/sqlite/events.go, internal/storage/sqlite/messages.go, internal/storage/sqlite/export.go.
package sqlite

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func TestPayloadDeduplicatedAndRestored(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	ctx, id := context.Background(), s.Session().ID
	text := strings.Repeat("long shared reasoning\n", 10000)
	for range 3 {
		if err := s.RecordEvent(ctx, id, runrecord.Event{Kind: "thinking", Thinking: text}); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.AppendMessage(ctx, id, modelapi.Message{Role: "assistant", Content: "answer", Reasoning: text}); err != nil {
		t.Fatal(err)
	}
	var count, bytesStored int
	if err := s.db.QueryRow("SELECT COUNT(*),SUM(length(body)) FROM payloads").Scan(&count, &bytesStored); err != nil {
		t.Fatal(err)
	}
	if count != 1 || bytesStored >= len(text)/4 {
		t.Fatalf("count=%d stored=%d", count, bytesStored)
	}
	page, _, err := s.Timeline(ctx, id, 0, 10)
	if err != nil || len(page) != 4 || page[0].Event.Thinking != text || page[3].Message.Content != "answer" || page[3].Message.Reasoning != "" {
		t.Fatal("round trip failed", err)
	}
	messages, err := s.ContextMessages(ctx, id, 0, 1<<20, true)
	if err != nil || len(messages) != 1 || messages[0].Reasoning != text {
		t.Fatal("model history lost full reasoning", err)
	}
	var output bytes.Buffer
	if err := s.Export(ctx, id, &output); err != nil {
		t.Fatal(err)
	}
	decoder := json.NewDecoder(&output)
	for range 4 {
		var record struct{ Body json.RawMessage }
		if err := decoder.Decode(&record); err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(record.Body, []byte("long shared reasoning")) {
			t.Fatal("export omitted payload")
		}
	}
}

func TestPayloadKeepsNativeArgumentBytes(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	ctx, id := context.Background(), s.Session().ID
	args := json.RawMessage(fmt.Sprintf(`{"path":"a.go","content":%q}`, strings.Repeat("source", 10000)))
	call := toolapi.Call{ID: "call_payload", Name: "write_file", Arguments: args}
	if err := s.AppendMessage(ctx, id, modelapi.Message{Role: "assistant", Calls: []toolapi.Call{call}}); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordEvent(ctx, id, runrecord.Event{Kind: "tool_succeeded", Call: &call, CallID: call.ID}); err != nil {
		t.Fatal(err)
	}
	page, _, err := s.Timeline(ctx, id, 0, 10)
	if err != nil || len(page) != 1 || page[0].Event.Call == nil || bytes.Contains(page[0].Event.Call.Arguments, []byte("source")) {
		t.Fatal("display loaded hidden source arguments", err)
	}
	messages, err := s.ContextMessages(ctx, id, 0, 1<<20, true)
	if err != nil || len(messages) != 1 || !bytes.Equal(messages[0].Calls[0].Arguments, args) {
		t.Fatal("canonical native argument bytes changed", err)
	}
	var exported bytes.Buffer
	if err = s.Export(ctx, id, &exported); err != nil || !bytes.Contains(exported.Bytes(), []byte("sourcesource")) {
		t.Fatal("export lost native argument content", err)
	}
	if _, err := s.db.Exec("UPDATE payloads SET body=x'00'"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ContextMessages(ctx, id, 0, 1024*1024, true); err == nil {
		t.Fatal("corrupt payload accepted")
	}
}
