// Purpose: Resolve documented checkpoint identities without guessing arbitrary serving aliases.
// Pattern: Fixed catalogue and boundary parser.
// Usage: Provider transport and configuration resolve a model or an explicit profile ID.
// Related: internal/provider/profile/qwen.go, internal/config/profiles.go, internal/provider/diagnostic/control.go.
package profile

import (
	"fmt"
	modelconn "github.com/keonho-kim/v-code/internal/provider/connection"
	"strings"
)

const Generic = "generic"
const reviewed = "2026-10-13"

// TotalB is the publisher-reported total architecture size, never active MoE size.
// Released preserves source precision: YYYY-MM or YYYY-MM-DD, never update time.
// An excluded record may leave an unconfirmed release empty; it cannot become eligible.
type Profile struct {
	Effort       EffortControl
	ID, Family   string
	Models       []string
	Released     string
	TotalB       float64
	TemplateKeys []string
	VLLMOnly     bool
	Source       string
}

func (p Profile) eligible() bool {
	return p.Released >= "2026-03" && p.Released <= reviewed && p.TotalB > 0 && p.TotalB <= 300
}

func catalogue() []Profile {
	var result []Profile
	for _, family := range [][]Profile{glm(), qwen(), nemotron(), deepseek(), exaone(), gemma()} {
		result = append(result, family...)
	}
	return result
}

// Resolve leaves the served request ID unchanged. Only registered exact identities match.
// A generic override deliberately disables automatic family specialization.
func Resolve(model, override string) (Profile, error) {
	generic := Profile{ID: Generic}
	if override == Generic {
		return generic, nil
	}
	var inferred, selected Profile
	for _, p := range catalogue() {
		if p.ID == override {
			selected = p
		}
		for _, name := range p.Models {
			if strings.EqualFold(model, name) {
				inferred = p
				break
			}
		}
	}
	if override != "" {
		if selected.ID == "" {
			return generic, fmt.Errorf("unknown model profile %q", override)
		}
		if !selected.eligible() {
			return generic, fmt.Errorf("model profile %q is outside the release or total-parameter limits", override)
		}
		if inferred.ID != "" && inferred.ID != selected.ID {
			return generic, fmt.Errorf("model identity conflicts with profile %q", override)
		}
		return selected, nil
	}
	if inferred.eligible() {
		return inferred, nil
	}
	return generic, nil
}

// ThinkingKeys exposes only documented controls for the selected serving dialect.
func (p Profile) ThinkingKeys(provider modelconn.ProviderKind) []string {
	if !p.eligible() || provider != modelconn.VLLM && provider != modelconn.SGLang || p.VLLMOnly && provider != modelconn.VLLM {
		return nil
	}
	return p.TemplateKeys
}
