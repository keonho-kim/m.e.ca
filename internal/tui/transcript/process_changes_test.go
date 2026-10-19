// Purpose: Show observed shell deletions without exposing raw JSON or invented file effects.
// Pattern: Presentation contract test.
// Usage: go test ./internal/tui/transcript -run TestCommandChanges.
// Related: internal/tui/transcript/process_changes.go.
package transcript

import (
	"github.com/keonho-kim/v-code/internal/agent/run"
	"github.com/keonho-kim/v-code/internal/agent/tool"
	"strings"
	"testing"
)

func TestCommandChanges(t *testing.T) {
	f := Formatter{Redact: func(s string) string { return s }}
	event := run.Event{Kind: "tool_failed", Text: "partial command failure", Call: &tool.Call{Name: "execute_command", Arguments: []byte(`{"command":"rm a outside"}`)}, Output: []byte(`{"ExitCode":1,"Status":"exited"}`), Operation: &tool.Operation{Changes: []tool.FileChange{{Action: "delete", Path: "/p/a", BeforeDigest: "abc", AfterDigest: "missing"}}}}
	shown := f.Tool(event)
	if !strings.Contains(shown, "삭제 확인: /p/a") || !strings.Contains(shown, "종료 코드: 1") || strings.Contains(shown, "BeforeDigest") {
		t.Fatal(shown)
	}
}
