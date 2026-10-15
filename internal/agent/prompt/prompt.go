// Purpose: Compose the work instructions with the project-provided development guidance.
// Pattern: Pure Function.
// Usage: The run owner supplies one project instruction snapshot.
// Related: internal/agent/prompt/prompt_test.go, internal/agent/run/context.go.
package prompt

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const MaxInstructionsBytes = 16 * 1024 * 1024

func Compose(base, instructions string) (string, error) {
	if err := checkInstructions(instructions); err != nil {
		return "", err
	}
	if err := CheckTemplate(base); err != nil {
		return "", err
	}
	text := base + "\n\nProject instructions from project-root AGENTS.md:\n" + instructions
	if instructions == "" {
		text += "(No project-root AGENTS.md was found.)"
	}

	return text, nil
}

func checkInstructions(instructions string) error {
	if len(instructions) > MaxInstructionsBytes || !utf8.ValidString(instructions) || strings.ContainsRune(instructions, 0) {
		return fmt.Errorf("project AGENTS.md exceeds the instruction limit or has invalid encoding")
	}
	return nil
}
