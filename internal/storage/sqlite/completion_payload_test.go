// Purpose: Measure duplicate completion and thinking representations in SQLite.
// Pattern: Persistence regression test.
// Usage: go test ./internal/storage/sqlite.
// Related: internal/agent/run/commands.go, internal/agent/run/generate.go.
package sqlite

import (
	"encoding/base64"
	"encoding/json"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"math/rand"
	"strings"
	"testing"
)

func TestAsyncCompletionPayloadDuplicates(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	raw := make([]byte, 24<<10)
	rand.New(rand.NewSource(1)).Read(raw)
	output, _ := json.Marshal(map[string]any{"Stdout": base64.StdEncoding.EncodeToString(raw), "Status": "exited", "ExitCode": 0})
	call := toolapi.Call{ID: "call_one", Name: "execute_command", Arguments: json.RawMessage(`{"command":"fixture"}`)}

	completed := modelapi.Message{Role: "system", Content: string(output), ToolCallID: call.ID, ToolName: call.Name}
	terminal := runrecord.Event{Kind: "command_finished", CallID: call.ID, Call: &call, Output: output}
	if err := s.Commit(t.Context(), s.Session().ID, []runrecord.Entry{{Message: &completed}, {Event: &terminal}}); err != nil {
		t.Fatal(err)
	}
	var count, rawBytes, stored int
	if err := s.db.QueryRowContext(t.Context(), "SELECT COUNT(*),SUM(raw_bytes),SUM(length(body)) FROM payloads").Scan(&count, &rawBytes, &stored); err != nil {
		t.Fatal(err)
	}
	t.Logf("output=%d payloads=%d raw-bytes=%d stored-bytes=%d", len(output), count, rawBytes, stored)
	if count > 1 {
		t.Fatal("completion wrapper creates another immutable payload for the same command output")
	}
}
func TestSmallThinkingCopies(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	text := strings.Repeat("r", 8<<10)
	event := runrecord.Event{Kind: "thinking", Thinking: text}
	message := modelapi.Message{Role: "assistant", Content: "done", Reasoning: text}
	if err := s.Commit(t.Context(), s.Session().ID, []runrecord.Entry{{Event: &event}, {Message: &message}}); err != nil {
		t.Fatal(err)
	}
	var count, bytes int
	if err := s.db.QueryRowContext(t.Context(), "SELECT COUNT(*),SUM(raw_bytes) FROM payloads").Scan(&count, &bytes); err != nil {
		t.Fatal(err)
	}
	if count != 1 || bytes > len(text)+2 {
		t.Fatalf("thinking is duplicated: payloads=%d raw=%d", count, bytes)
	}
	messages, err := s.ContextMessages(t.Context(), s.Session().ID, 0, 1<<20, false, nil)
	if err != nil || len(messages) != 1 || messages[0].Reasoning != text {
		t.Fatalf("reasoning did not restore: %v", err)
	}
	page, _, err := s.Timeline(t.Context(), s.Session().ID, 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, r := range page {
		if r.Event.Kind == "thinking" {
			found = r.Event.Thinking == text
		}
	}
	if !found {
		t.Fatal("thinking history did not restore")
	}
}
