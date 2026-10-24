// Purpose: Verify tool displays retain useful results without runtime metadata or write-body duplication.
// Pattern: Presentation contract tests.
// Usage: go test ./internal/tui/transcript -run TestCompactTool.
// Related: internal/tui/transcript/tools.go.
package transcript

import (
	"github.com/keonho-kim/meca/internal/agent/run"
	"github.com/keonho-kim/meca/internal/agent/tool"
	"strings"
	"testing"
)

func TestCompactToolShellRead(t *testing.T) {
	f := Formatter{Redact: func(s string) string { return s }}
	event := run.Event{Kind: "tool_succeeded", Call: &tool.Call{Name: "execute_command", Arguments: []byte(`{"command":"cat a"}`)}, Output: []byte(`{"Stdout":"hello","ExitCode":0,"Status":"exited","Truncated":true}`)}
	shown := f.Tool(event)
	for _, hidden := range []string{"Editable", "FinalNewline", "SizeBytes", "NextOffset", "internal reason", "limit:", "offset:"} {
		if strings.Contains(shown, hidden) {
			t.Fatal(shown)
		}
	}
	if !strings.Contains(shown, "hello") || !strings.Contains(shown, "일부") {
		t.Fatal(shown)
	}
}
func TestCompactToolWrite(t *testing.T) {
	f := Formatter{Redact: func(s string) string { return s }}
	shown := f.Tool(run.Event{Kind: "tool_succeeded", Text: "Completed", Call: &tool.Call{Name: "write_file", Arguments: []byte(`{"path":"a","content":"large source body","format_policy":"preserve"}`)}, Output: []byte(`{"path":"a","changed":true}`), ChangePreview: "--- /dev/null\n+++ a\n@@ -0,0 +1,1 @@\n+large source body\n"})
	if strings.Count(shown, "large source body") != 1 || strings.Contains(shown, "format_policy") || !strings.Contains(shown, "파일 저장 완료") {
		t.Fatal(shown)
	}
}
