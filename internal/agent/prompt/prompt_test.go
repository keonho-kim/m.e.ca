// Purpose: Verify project instructions remain in every fresh bounded model context.
// Pattern: Prompt contract tests.
// Usage: go test ./internal/agent/prompt.
// Related: internal/agent/prompt/prompt.go.
package prompt

import (
	"strings"
	"testing"
)

func TestProjectRulesAndBounds(t *testing.T) {
	text, err := Compose(Defaults().System, "Use focused Go tests.", false)
	if err != nil || !strings.Contains(text, "project-root AGENTS.md") || !strings.Contains(text, "Use focused Go tests.") {
		t.Fatalf("instructions: %s %v", text, err)
	}
	if _, err := Compose(Defaults().System, strings.Repeat("a", MaxInstructionsBytes+1), false); err == nil {
		t.Fatal("silently truncated project instructions")
	}
}
