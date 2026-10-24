// Purpose: Format run duration and the bounded compression header indicator.
// Pattern: Pure presentation values.
// Usage: Session projects its worker state into every view through Activity.
// Related: internal/tui/view/tokens.go, internal/tui/view/input.go.
package view

import (
	"fmt"
	"github.com/keonho-kim/meca/internal/tui/locale"
	"strings"
	"time"
)

type Activity struct {
	Language             locale.Language
	Elapsed              time.Duration
	TokensPerSecond      *float64
	Running, Compressing bool
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
		label += " · " + a.compressionLabel()
	}
	return label
}

func (a Activity) compressionLabel() string {
	dots := strings.Repeat(".", max(1, min(3, a.Dots)))
	return fmt.Sprintf("%s %-3s", a.Language.Text("Compress Memory"), dots)
}
