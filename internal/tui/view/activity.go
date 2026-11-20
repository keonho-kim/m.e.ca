// Purpose: Format run duration, compaction status and the active model attempt’s timing.
// Pattern: Pure presentation values.
// Usage: Session projects its worker state into every view through Activity.
// Related: internal/tui/view/tokens.go, internal/tui/view/input.go.
package view

import (
	"fmt"
	"strings"
	"time"

	"github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/tui/locale"
)

type Activity struct {
	Language             locale.Language
	MemoryUsage          string
	Elapsed              time.Duration
	Timing               *record.ModelActivity
	TimingAgent          string
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

func (a Activity) speedLabel(now time.Time) string {
	m := a.Timing
	if m == nil {
		return ""
	}
	label := a.Language.Text("TPS") + " —"
	if m.FirstOutput.IsZero() && !m.Finished {
		label = fmt.Sprintf("%s %ds", a.Language.Text("Waiting for output"), max(0, int(now.Sub(m.Started).Seconds())))
	} else if rate := m.Speed.Rate(); rate != nil {
		source := "Received TPS"
		if m.Speed.Source == "provider" {
			source = "Generation TPS"
		}
		label = fmt.Sprintf("%s %.1f", a.Language.Text(source), *rate)
	}
	if a.TimingAgent != "" {
		label = a.TimingAgent + " " + label
	}
	return " · " + label
}
