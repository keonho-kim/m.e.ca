// Purpose: Resolve selectable effort values from declared native or reviewed profile controls.
// Pattern: Simple Module.
// Usage: Model menus use EffortValues without interpreting profile rules themselves.
// Related: internal/provider/selection/validate.go, internal/provider/profile/effort.go.
package selection

import (
	modelconn "github.com/keonho-kim/v-code/internal/provider/connection"
	"github.com/keonho-kim/v-code/internal/provider/profile"
)

func EffortValues(selected modelconn.Selection) []string {
	if selected.Details != nil && selected.Details.Reasoning != nil {
		if selected.Thinking != nil {
			return selected.Details.Reasoning.Levels
		}
		return nil
	}
	if !selected.ThinkingSupported {
		return nil
	}
	resolved, err := profile.Resolve(selected.Model, selected.Profile)
	if err != nil {
		return nil
	}
	return resolved.EffortValues(selected.Connection.Kind)
}
