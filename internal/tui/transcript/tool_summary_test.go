// Purpose: Verify collapsed results keep failure reasons and redact secrets before shortening.
// Pattern: Presentation contract tests.
// Usage: go test ./internal/tui/transcript -run TestToolSummary.
// Related: internal/tui/transcript/tool_summary.go, internal/tui/transcript/events.go.
package transcript

import (
	"strings"
	"testing"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/tool"
)

func TestToolSummaryFailures(t *testing.T) {
	f := Formatter{Redact: func(s string) string { return strings.ReplaceAll(s, "secret-value", "[redacted]") }}
	for _, kind := range []string{"tool_failed", "tool_interrupted", "denied"} {
		event := runrecord.Event{Kind: kind, Text: "target changed: secret-value", Call: &tool.Call{Name: "edit_file", Arguments: []byte(`{"path":"a.txt"}`)}, Output: []byte(`{"changed":false}`)}
		block := f.Event(event)[0]
		if !strings.Contains(block.Summary, "target changed") || strings.Contains(block.Summary, "secret-value") || !strings.Contains(block.Summary, "접힘") || strings.Contains(block.Summary, "변경 없음") {
			t.Fatalf("misleading collapsed failure: %q", block.Summary)
		}
	}
}
