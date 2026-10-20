// Purpose: Resolve effort choices from server metadata or the selected serving API.
// Pattern: Simple Module.
// Usage: Model menus use EffortValues without interpreting wire fields.
// Related: internal/provider/connection/reasoning_options.go, internal/provider/selection/validate.go.
package selection

import "github.com/keonho-kim/v-code/internal/provider/connection"

func EffortValues(selected connection.Selection) []string {
	options := connection.ReasoningOptions(selected.Connection.Kind, selected.Details)
	return options.Levels
}
