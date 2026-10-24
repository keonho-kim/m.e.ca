// Purpose: Format final request measurements as a muted completion line in the selected UI language.
// Pattern: Presentation projection.
// Usage: Formatter.Event renders live and archived finished checkpoints through runSummary.
// Related: internal/agent/run/statistics.go, internal/tui/transcript/events.go, internal/tui/viewport/alignment.go.
package transcript

import (
	"fmt"
	"strconv"

	"github.com/keonho-kim/meca/internal/agent/run"
	"github.com/keonho-kim/meca/internal/tui/viewport"
)

func (f Formatter) runSummary(s run.Statistics) viewport.Block {
	seconds := s.ElapsedMillis / 1000
	duration := fmt.Sprintf("%02d:%02d:%02d", seconds/3600, seconds/60%60, seconds%60)
	input, output, rate := "—", "—", "—"
	if s.Usage != nil {
		input, output = strconv.FormatInt(s.Usage.Input, 10), strconv.FormatInt(s.Usage.Output, 10)
		if s.GenerationMillis > 0 {
			rate = fmt.Sprintf("%.1f", float64(s.Usage.Output)*1000/float64(s.GenerationMillis))
		}
	}
	text := f.Language.Text("Worked for") + " " + duration +
		" | " + f.Language.Text("INPUT") + " " + input +
		" | " + f.Language.Text("OUTPUT") + " " + output +
		" | " + f.Language.Text("TPS") + " " + rate
	return viewport.Block{Text: text + "\n\n", Thinking: true, Centered: true}
}
