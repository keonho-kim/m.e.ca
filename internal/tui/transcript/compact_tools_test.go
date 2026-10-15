// Purpose: Verify tool displays retain useful results without runtime metadata or write-body duplication.
// Pattern: Presentation contract tests.
// Usage: go test ./internal/tui/transcript -run TestCompactTool.
// Related: internal/tui/transcript/tools.go.
package transcript

import (
	"github.com/keonho-kim/v-code/internal/agent/run"
	"github.com/keonho-kim/v-code/internal/agent/tool"
	"strings"
	"testing"
)

func TestCompactToolRead(t *testing.T) {
	f := Formatter{Redact: func(s string) string { return s }}
	event := run.Event{Kind: "tool_succeeded", Call: &tool.Call{Name: "read_file", Arguments: []byte(`{"path":"a","offset":0,"limit":4096,"reason":"internal reason"}`)}, Output: []byte(`{"Content":"1#1234abcd | hello","Path":"a","Editable":true,"FinalNewline":true,"SizeBytes":6,"Truncated":true,"NextOffset":6}`)}
	shown := f.Tool(event)
	for _, hidden := range []string{"Editable", "FinalNewline", "SizeBytes", "NextOffset", "internal reason", "limit:", "offset:"} {
		if strings.Contains(shown, hidden) {
			t.Fatal(shown)
		}
	}
	if !strings.Contains(shown, "1 | hello") || !strings.Contains(shown, "일부") {
		t.Fatal(shown)
	}
}
func TestCompactToolWrite(t *testing.T) {
	f := Formatter{Redact: func(s string) string { return s }}
	shown := f.Tool(run.Event{Kind: "tool_succeeded", Text: "Completed", Call: &tool.Call{Name: "write_file", Arguments: []byte(`{"path":"a","content":"large source body","format_policy":"preserve"}`)}, Output: []byte(`{"path":"a","changed":true}`)})
	if strings.Contains(shown, "large source body") || strings.Contains(shown, "format_policy") || !strings.Contains(shown, "파일 저장 완료") {
		t.Fatal(shown)
	}
}
