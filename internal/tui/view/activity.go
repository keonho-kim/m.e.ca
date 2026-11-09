// Purpose: Format run duration and the bounded compaction header indicator.
// Pattern: Pure presentation values.
// Usage: Session projects its worker state into every view through Activity.
// Related: internal/tui/view/tokens.go, internal/tui/view/input.go.
package view

import (
	"fmt"
	"strings"
	"time"

	"github.com/keonho-kim/meca/internal/tui/locale"
)

type Activity struct {
	Language             locale.Language
	MemoryUsage          string
	Elapsed              time.Duration
	TokensPerSecond      *float64
	Running, Compressing bool
	ShowCost             bool
	Dots                 int
}

func (a Activity) Status() string {
	if a.Running {
		return a.Language.Text("RUNNING")
	}
	return a.Language.Text("IDLE")
}
func (a Activity) Label() string {
	label := a.Language.Text("STATUS") + " " + a.Status()
	if a.Running {
		seconds := int64(max(0, a.Elapsed) / time.Second)
		label += fmt.Sprintf(" %02d : %02d : %02d", seconds/3600, seconds/60%60, seconds%60)
	}
	if a.Compressing {
		label += " · " + a.indicatorLabel()
	}
	return label
}

func (a Activity) indicatorLabel() string {
	dots := strings.Repeat(".", max(1, min(3, a.Dots)))
	label := a.Language.Text("Compress Memory")
	return fmt.Sprintf("%s %-3s", label, dots)
}
