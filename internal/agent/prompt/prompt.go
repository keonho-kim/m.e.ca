// Purpose: Compose project-root instructions with runtime invariants and a small sequential tool protocol.
// Pattern: Pure Function.
// Usage: Rebuilt from one instruction snapshot for every fresh Ralph iteration.
// Related: internal/agent/prompt/prompt_test.go, internal/agent/run/context.go.
package prompt

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const MaxInstructionsBytes = 32 * 1024

func Compose(base, instructions string, planning bool) (string, error) {
	if len(instructions) > MaxInstructionsBytes || !utf8.ValidString(instructions) || strings.ContainsRune(instructions, 0) {
		return "", fmt.Errorf("project AGENTS.md exceeds the instruction limit or has invalid encoding")
	}
	if err := CheckTemplate(base); err != nil {
		return "", err
	}
	text := base + "\n\nProject instructions from project-root AGENTS.md:\n" + instructions
	if instructions == "" {
		text += "(No project-root AGENTS.md was found.)"
	}
	if planning {
		text += "\nPlanning only: scoped reads and searches are permitted. Never change files or execute commands. For DONE, put the final Plan as JSON on the following lines: {\"objective\":\"...\",\"steps\":[\"...\"]}."
	}
	return text, nil
}
