// Purpose: Determine whether the serving API offers both Thinking ON and OFF requests.
// Pattern: Pure function.
// Usage: Setup exposes request choices; explicit diagnostics verify actual model behavior.
// Related: internal/provider/connection/reasoning_options.go, internal/provider/diagnostic/check.go.
package diagnostic

import "github.com/keonho-kim/meca/internal/provider/connection"

func ThinkingControl(kind connection.ProviderKind, info *connection.ModelInfo) bool {
	options := connection.ReasoningOptions(kind, info)
	return options.Enabled() && options.Off
}
