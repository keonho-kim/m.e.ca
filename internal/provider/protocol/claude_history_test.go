// Purpose: Keep Claude block order while dropping thinking whose earlier conversation changed.
// Pattern: Stateless request and response continuation contract tests.
// Usage: go test ./internal/provider/protocol -run TestClaudeHistory.
// Related: internal/provider/protocol/claude_request.go, internal/provider/protocol/claude/continuation.go.
package protocol

import (
	"encoding/json"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/protocol/claude"
	"strings"
	"testing"
)

func TestClaudeHistoryBindsPrefixAndKeepsBlocks(t *testing.T) {
	request := modelapi.Request{ModelID: "alias", MaxOutput: 32}
	first := []Message{{Role: "system", Content: "rules"}, {Role: "user", Content: "objective"}, {Role: "user", Content: "temporary team observation"}}
	_, body, err := singleBody(connection.Claude, "https://example.com/v1", nil, request, first, Controls{})
	if err != nil {
		t.Fatal(err)
	}
	raw := []byte(`[{"type":"text","text":"before"},{"type":"thinking","thinking":"","signature":"signed","opaque":{"large":9007199254740993}},{"type":"text","text":"after"}]`)
	state, err := body.BindContinuation(raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range []string{"none", "system", "compaction", "observation", "tools"} {
		t.Run(change, func(t *testing.T) {
			messages := append([]Message(nil), first...)
			selected := request
			switch change {
			case "system":
				messages[0].Content = "new rules"
			case "compaction":
				messages = []Message{{Role: "system", Content: "rules"}, {Role: "user", Content: "summary"}}
			case "observation":
				messages = messages[:2]
			case "tools":
				selected.Tools = []toolapi.Definition{{Name: "list_command_runs", Description: "List owned commands"}}
			}
			messages = append(messages, Message{Role: "assistant", Content: "beforeafter", continuation: state}, Message{Role: "user", Content: "continue"})
			_, encoded, err := Encode(connection.Claude, "https://example.com/v1", nil, selected, messages, Controls{})
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(encoded), `"signature":"signed"`) != (change == "none") {
				t.Fatalf("stale thinking retained or valid thinking lost: %s", encoded)
			}
			var sent claude.Request
			if err := json.Unmarshal(encoded, &sent); err != nil {
				t.Fatal(err)
			}
			for _, m := range sent.Messages {
				if m.Role != "assistant" {
					continue
				}
				if m.Content[0].Text != "before" || m.Content[len(m.Content)-1].Text != "after" {
					t.Fatalf("text blocks coalesced: %+v", m.Content)
				}
			}
			if change == "none" && !strings.Contains(string(encoded), "9007199254740993") {
				t.Fatal("unknown signed metadata changed")
			}
		})
	}
}
