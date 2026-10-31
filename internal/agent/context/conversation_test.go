// Purpose: Verify native history pairing and safe preservation of interrupted or compacted calls.
// Pattern: Context projection contract tests.
// Usage: go test ./internal/agent/context.
// Related: internal/agent/context/conversation.go, internal/agent/context/request.go.
package context

import (
	"encoding/json"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
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
				if diff := cmp.Diff(history[i], message, cmpopts.IgnoreUnexported(modelapi.Message{})); diff != "" {
					t.Fatalf("native history (-want +got):\n%s", diff)
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

func TestUnfinishedDraftIsQuotedWithoutReasoning(t *testing.T) {
	history := []modelapi.Message{
		{Role: "user", Content: "Create the endpoint"},
		{Role: "assistant", Content: "The endpoint is implemented", Reasoning: "unfinished-thought", Incomplete: true},
		{Role: "system", Content: "Request interrupted", Reasoning: "runtime-thought"},
		{Role: "user", Content: "Continue with authentication"},
	}
	before, _ := json.Marshal(history)
	request, err := Build(Input{Messages: history, Objective: "Continue with authentication", Redact: func(s string) string { return s }, MaxBytes: 10000})
	if err != nil {
		t.Fatal(err)
	}
	draft := request.Messages[2]
	if draft.Role != "user" || draft.Incomplete || draft.Reasoning != "" || !strings.Contains(draft.Content, "Unfinished assistant draft") || !strings.Contains(draft.Content, "The endpoint is implemented") {
		t.Fatalf("unsafe draft projection: %+v", draft)
	}
	body, _ := json.Marshal(request.Messages)
	for _, thought := range []string{"unfinished-thought", "runtime-thought"} {
		if strings.Contains(string(body), thought) {
			t.Fatalf("replayed %s", thought)
		}
	}
	if request.Messages[len(request.Messages)-1].Content != "Continue with authentication" {
		t.Fatal("newest instruction no longer last")
	}
	after, _ := json.Marshal(history)
	if string(before) != string(after) {
		t.Fatal("projection changed archived history")
	}
}
