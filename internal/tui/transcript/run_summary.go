// Purpose: Format final request measurements as a muted completion line in the selected UI language.
// Pattern: Presentation projection.
// Usage: Formatter.Event renders live and archived finished checkpoints through runSummary.
// Related: internal/agent/run/statistics.go, internal/tui/transcript/events.go, internal/tui/viewport/alignment.go.
package transcript

import (
	"fmt"
	"strconv"
	"time"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/tui/viewport"
)

func (f Formatter) runSummary(s runrecord.Statistics) viewport.Block {
	seconds := s.ElapsedMillis / 1000
	duration := fmt.Sprintf("%02d:%02d:%02d", seconds/3600, seconds/60%60, seconds%60)
	input, output, rate := "—", "—", ""
	if s.Usage != nil {
		input, output = strconv.FormatInt(s.Usage.Input, 10), strconv.FormatInt(s.Usage.Output, 10)
		if measured := (runrecord.Throughput{Output: s.Usage.Output, Duration: time.Duration(s.GenerationMillis) * time.Millisecond}).Rate(); measured != nil {
			rate = fmt.Sprintf("%.1f", *measured)
		}
	}
	text := f.Language.Text("Worked for") + " " + duration +
		" | " + f.Language.Text("INPUT") + " " + input +
		" | " + f.Language.Text("OUTPUT") + " " + output
	if s.Usage != nil && s.Usage.Cost != nil {
		text += " | " + f.Language.Cost(s.Usage.Cost)
	}
	if rate != "" {
		text += " | " + f.Language.Text("TPS") + " " + rate
	}
	return viewport.Block{Text: text + "\n\n", Thinking: true, Centered: true}
}
