// Purpose: Verify native history pairing and safe preservation of interrupted or compacted calls.
// Pattern: Context projection contract tests.
// Usage: go test ./internal/agent/context.
// Related: internal/agent/context/conversation.go, internal/agent/context/request.go.
package context

import (
	"encoding/json"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"strings"
	"testing"
)

func TestNativeHistoryPreservesPairsAndQuotesOrphans(t *testing.T) {
	call := toolapi.Call{ID: "c", Name: "inspect_fixture", Arguments: json.RawMessage(`{"path":"a"}`)}
	messages := []modelapi.Message{
		{Role: "assistant", Reasoning: "inspect a", Calls: []toolapi.Call{call}},
		{Role: "tool", ToolCallID: "c", ToolName: "inspect_fixture", Content: "observed"},
	}
	for _, history := range [][]modelapi.Message{messages, messages[:1], messages[1:]} {
		request, err := Build(Input{Messages: history, Redact: func(s string) string { return s }, MaxBytes: 10000})
		if err != nil {
			t.Fatal(err)
		}
		projected := request.Messages[1:]
		for i, message := range projected {
			if len(history) == 2 {
				if message.Role != history[i].Role || message.ToolCallID != history[i].ToolCallID || message.Reasoning != history[i].Reasoning {
					t.Fatalf("native history lost: %+v", projected)
				}
			} else if message.Role != "user" || len(message.Calls) != 0 || !strings.Contains(message.Content, "Earlier notes and observations") {
				t.Fatalf("orphan replayed: %+v", message)
			}
		}
		if len(history) == 1 && len(history[0].Calls) > 0 && !strings.Contains(projected[0].Content, "inspect a") {
			t.Fatal("orphan reasoning discarded")
		}
	}
}

func TestActiveFollowupFollowsOlderUserMessages(t *testing.T) {
	request, err := Build(Input{Objective: "Implement the remaining endpoint", Messages: []modelapi.Message{{Role: "user", Content: "Original implementation request"}}, Redact: func(s string) string { return s }, MaxBytes: 10000})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(request.Messages[len(request.Messages)-1].Content, "Implement the remaining endpoint") {
		t.Fatalf("active request preceded history: %+v", request.Messages)
	}
}
