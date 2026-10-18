// Purpose: Select the session's immutable prompt snapshot.
// Pattern: Simple Module.
// Usage: The main agent loads system and development guidance before generation.
// Related: internal/agent/prompt/defaults.go.
package run

import "github.com/keonho-kim/v-code/internal/agent/prompt"

func (r Runner) templates() prompt.Templates {
	if r.Prompts != nil {
		return *r.Prompts
	}
	return prompt.Defaults()
}
