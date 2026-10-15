// Purpose: Compose internal role instructions without the main agent dialogue protocol.
// Pattern: Pure prompt composition.
// Usage: Conclave uses Internal with project constraints.
// Related: internal/agent/prompt/prompt.go, internal/agent/prompt/defaults.go.
package prompt

func Internal(base, instructions string) (string, error) {
	if err := CheckTemplate(base); err != nil {
		return "", err
	}
	if err := checkInstructions(instructions); err != nil {
		return "", err
	}
	text := base + "\n\nProject constraints from project-root AGENTS.md:\n" + instructions
	text += "\n\nInternal role contract: follow the current role's response format over project response-language or presentation preferences. Use English for internal descriptive fields. Preserve original paths, code, literals and quoted evidence exactly. Do not execute tools in this internal generation call. Runtime Scope, approval and verification remain mandatory."
	return text, nil
}
