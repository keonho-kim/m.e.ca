// Purpose: Report diagnostic failures through an independent fixed-message stderr path.
// Pattern: Nonrecursive fallback output.
// Usage: Missing reporter and writer failure paths emit no caller values or error text.
// Related: internal/diagnostics/reporter.go, internal/diagnostics/logger.go.
package diagnostics

import (
	"github.com/keonho-kim/meca/internal/diagnostics/catalog"
)

func emergency(event catalog.Event) {
	emergencyWrite([]byte("{\"level\":\"WARN\",\"msg\":\"" + string(event) + "\"}\n"))
}
