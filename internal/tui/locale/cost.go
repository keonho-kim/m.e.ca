// Purpose: Format received USD charges consistently across live headers and completed work summaries.
// Pattern: Pure presentation function.
// Usage: Terminal metrics render cost only for OpenRouter-reported measurements.
// Related: internal/tui/view/tokens.go, internal/tui/transcript/run_summary.go.
package locale

import (
	"fmt"
	"strings"
)

// Cost keeps unavailable charges distinct from free calls without estimating a price.
func (l Language) Cost(usd *float64) string {
	value := "—"
	if usd != nil {
		value = "$" + strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.6f", *usd), "0"), ".")
		if *usd > 0 && *usd < 0.000001 {
			value = "<$0.000001"
		}
	}
	return l.Text("TOTAL COST") + " " + value
}
