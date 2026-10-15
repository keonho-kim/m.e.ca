// Purpose: Keep supplied work prompts free of agent-orchestration explanations.
// Pattern: Prompt contract regression tests.
// Usage: go test ./internal/agent/prompt -run TestTaskPrompts.
// Related: internal/agent/prompt/defaults.go, internal/agent/prompt/prompt.go.
package prompt

import (
	"strings"
	"testing"
)

func TestTaskPromptsDoNotDescribeHarness(t *testing.T) {
	templates := Defaults()
	for name, text := range templates.Files() {
		for _, term := range []string{"harness", "checkpoint", "runtime", "conclave", "completion_criteria", "fixed criteria"} {
			if strings.Contains(strings.ToLower(*text), term) {
				t.Errorf("%s exposes orchestration term %q", name, term)
			}
		}
	}
	system, err := Compose("Work on the request.", "Keep the API.")
	if err != nil || !strings.HasPrefix(system, "Work on the request.") || !strings.Contains(system, "Keep the API.") {
		t.Fatal("lost supplied work instructions")
	}
	if strings.Contains(system, "completion review") || strings.Contains(system, "native function") {
		t.Fatal("hidden orchestration instructions appended")
	}
}
