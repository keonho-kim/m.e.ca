// Purpose: Keep distinct admitted tools visible when providers reuse a call identity.
// Pattern: Pure single-mode presentation regression.
// Usage: go test ./internal/tui/session -run TestReusedCallID.
// Related: internal/tui/session/tool_activity.go, internal/agent/tool/call.go.
package session

import (
	"strings"
	"testing"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/tui/transcript"
)

func TestReusedCallIDKeepsDistinctExecutionsVisible(t *testing.T) {
	f := transcript.Formatter{Redact: func(text string) string { return text }}
	activity := toolActivity{commands: []toolapi.CommandRun{{ID: "cmd_old", CallID: "reused", ExecutionID: "exec_old", Command: "printf OLD_BACKGROUND", Status: "running"}}}
	call := toolapi.Call{ID: "reused", ExecutionID: "exec_new", Name: "write_file", Arguments: []byte(`{"path":"NEW_PENDING_FILE","content":"x"}`)}
	activity.start(&runrecord.Event{Kind: "tool_started", CallID: call.ID, Call: &call}, f)
	shown := activity.render(f, true)
	if !strings.Contains(shown, "NEW_PENDING_FILE") || !strings.Contains(shown, "OLD_BACKGROUND") {
		t.Fatalf("different execution was hidden by an older command: %q", shown)
	}
}
