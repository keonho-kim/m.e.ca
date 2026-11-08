// Purpose: Combine system principles, development guidance and project instructions.
// Pattern: Pure Function.
// Usage: The run owner supplies one project instruction snapshot.
// Related: internal/agent/prompt/prompt_test.go, internal/agent/run/context.go.
package prompt

import (
	"strings"
	"unicode/utf8"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

const MaxInstructionsBytes = 16 * 1024 * 1024

func Compose(base, instructions string) (string, error) {
	if err := checkInstructions(instructions); err != nil {
		return "", err
	}
	if err := CheckTemplate(base); err != nil {
		return "", err
	}
	if strings.TrimSpace(instructions) == "" {
		return base, nil
	}
	return base + "\n\nProject instructions from project-root AGENTS.md:\n" + instructions, nil
}

func checkInstructions(instructions string) error {
	if len(instructions) > MaxInstructionsBytes || !utf8.ValidString(instructions) || strings.ContainsRune(instructions, 0) {
		return faults.Errorf(diagcatalog.PromptProjectAGENTSMdExceedsTheInstructionLimit)
	}
	return nil
}

// Work combines system and development guidance with project constraints.
func Work(common, development, instructions string) (string, error) {
	if err := CheckTemplate(development); err != nil {
		return "", err
	}
	text, err := Compose(common, instructions)
	if err != nil {
		return "", err
	}
	return text + "\n\n" + development, nil
}
