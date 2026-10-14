// Purpose: Select the loaded prompt snapshot for each internal runtime phase.
// Pattern: Simple Module.
// Usage: Task requests and Conclave use these role-specific instructions.
// Related: internal/agent/prompt/internal.go, internal/agent/run/context.go.
package run

import "github.com/keonho-kim/v-code/internal/agent/prompt"

func (r Runner) templates() prompt.Templates {
	if r.Prompts != nil {
		return *r.Prompts
	}
	return prompt.Defaults()
}
func (s *task) phaseSystem() (string, error) {
	templates := s.runner.templates()
	var base string
	switch s.checkpoint.Phase {
	case "argument":
		base = templates.PFC
	case "grounding":
		base = templates.Grounding
	case "goal":
		base = templates.Goal
	default:
		return s.system, nil
	}
	text, err := prompt.Internal(base, s.runner.ProjectInstructions)
	if err != nil {
		return "", err
	}
	if s.planning {
		text += "\nPlanning only: scoped reads and searches are permitted; never change files or execute commands."
	}
	return text, nil
}
