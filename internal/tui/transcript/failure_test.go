// Purpose: Show a failed preparation once while retaining distinct command output and failure status.
// Pattern: Presentation contract regression.
// Usage: go test ./internal/tui/transcript -run TestPreparationFailure.
// Related: internal/tui/transcript/tools.go, internal/tui/transcript/details.go.
package transcript

import (
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/tool"
	"strings"
	"testing"
)

func TestPreparationFailureAppearsOnce(t *testing.T) {
	e := runrecord.Event{Kind: "tool_failed", Call: &tool.Call{Name: "execute_command", Arguments: []byte(`{"command":"bad command"}`)}, Text: "The command was not started", Output: []byte(`{"executed":false,"failure":{"code":"invalid_argument"}}`)}
	shown := (Formatter{Redact: func(s string) string { return s }}).Tool(e)
	if strings.Count(shown, e.Text) != 1 {
		t.Fatal(shown)
	}
}
