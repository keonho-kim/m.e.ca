// Purpose: Validate documented checkpoint-specific effort values for an identified serving dialect.
// Pattern: Simple Module.
// Usage: Configuration, provider requests and model settings use effort rules.
// Related: internal/provider/profile/profile.go, internal/provider/protocol/controls.go.
package profile

import (
	"fmt"
	modelconn "github.com/keonho-kim/v-code/internal/provider/connection"
	"slices"
)

type EffortKind uint8

const (
	ReasoningEffort EffortKind = iota + 1
	LowEffort
)

type EffortControl struct {
	Kind   EffortKind
	Values []string
}

func (p Profile) EffortValues(provider modelconn.ProviderKind) []string {
	if len(p.ThinkingKeys(provider)) == 0 {
		return nil
	}
	return p.Effort.Values
}

func (p Profile) CheckEffort(provider modelconn.ProviderKind, effort string) error {
	if effort == "" {
		return nil
	}
	if !slices.Contains(p.EffortValues(provider), effort) {
		return fmt.Errorf("Thinking effort %q is not documented for profile %s and provider %s", effort, p.ID, provider)
	}
	return nil
}
