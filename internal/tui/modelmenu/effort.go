// Purpose: Format the selected Thinking effort and availability for the model menu.
// Pattern: Presentation projection.
// Usage: Terminal model menus use EffortLabel.
// Related: internal/provider/selection/effort.go.
package modelmenu

import (
	"github.com/keonho-kim/v-code/internal/provider/connection"
	selectionpolicy "github.com/keonho-kim/v-code/internal/provider/selection"
	"strings"
)

func EffortLabel(selected connection.Selection) string {
	value := strings.ToUpper(selected.Effort())
	if value == "" {
		value = "DEFAULT"
	}
	label := "Thinking effort: " + value
	if len(selectionpolicy.EffortValues(selected)) == 0 {
		label += " (unavailable)"
	}
	return label
}
