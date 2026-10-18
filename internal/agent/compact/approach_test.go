// Purpose: Preserve stated working choices in the single compaction request without exposing private data.
// Pattern: Compaction input contract test.
// Usage: go test ./internal/agent/compact -run TestActiveApproach.
// Related: internal/agent/compact/summarize.go, internal/agent/run/memory.go.
package compact

import (
	"strings"
	"testing"
)

func TestActiveApproachIsPreservedAndRedacted(t *testing.T) {
	c := Controller{Prompt: "Summarize work", Settings: Defaults(), ActiveRequest: "이제 기본 구현을 해줘", ActiveApproach: []string{"Use local storage; secret=PRIVATE"}, Redact: func(s string) string { return strings.ReplaceAll(s, "PRIVATE", "[redacted]") }}
	messages, err := c.summaryMessages("Earlier scaffold", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	text := messages[len(messages)-1].Content
	if !strings.Contains(text, "이제 기본 구현을 해줘") || !strings.Contains(text, "Use local storage") || strings.Contains(text, "PRIVATE") {
		t.Fatalf("unsafe or missing work choices: %s", text)
	}
}
