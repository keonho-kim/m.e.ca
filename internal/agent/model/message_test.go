// Purpose: Verify unfinished drafts cannot carry executable native tool records.
// Pattern: Message boundary contract tests.
// Usage: go test ./internal/agent/model -run TestIncompleteMessage.
// Related: internal/agent/model/message.go, internal/agent/context/conversation.go.
package model

import (
	"encoding/json"
	"testing"

	"github.com/keonho-kim/v-code/internal/agent/tool"
)

func TestIncompleteMessage(t *testing.T) {
	for _, message := range []Message{
		{Role: "user", Incomplete: true},
		{Role: "system", Incomplete: true},
		{Role: "tool", Incomplete: true, ToolCallID: "call"},
		{Role: "assistant", Incomplete: true, ToolName: "write_file"},
		{Role: "assistant", Incomplete: true, Calls: []tool.Call{{ID: "call", Name: "write_file", Arguments: json.RawMessage(`{}`)}}},
	} {
		if err := message.Check(); err == nil {
			t.Fatalf("accepted invalid draft: %+v", message)
		}
	}
	if err := (Message{Role: "assistant", Content: "partial answer", Incomplete: true}).Check(); err != nil {
		t.Fatal(err)
	}
}
