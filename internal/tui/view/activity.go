// Purpose: Describe static run status and the bounded compression header indicator.
// Pattern: Pure presentation values.
// Usage: Session projects its worker state into every view through Activity.
// Related: internal/tui/view/tokens.go, internal/tui/view/input.go.
package view

import (
	"fmt"
	"strings"
)

type Activity struct {
	TokensPerSecond      *float64
	Running, Compressing bool
	Dots                 int
}

func (a Activity) Status() string {
	if a.Running {
		return "RUNNING"
	}
	return "IDLE"
}
func (a Activity) Label() string {
	label := "STATUS " + a.Status()
	if a.Running {
		label += " [*]"
	}
	if a.Compressing {
		dots := strings.Repeat(".", max(1, min(3, a.Dots)))
		label += fmt.Sprintf(" · Compress Memory %-3s", dots)
	}
	return label
}
