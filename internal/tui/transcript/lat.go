// Purpose: Render concise localized independent-review outcomes from durable events.
// Pattern: Presentation projection.
// Usage: History and live terminal events share this formatter.
// Related: internal/tui/transcript/events.go, internal/tui/session/runtime_events.go.
package transcript

import (
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/tui/viewport"
)

func (f Formatter) latReview(event runrecord.Event) viewport.Block {
	label := "LAT: Review feedback"
	switch event.Kind {
	case "lat_clean":
		label = "LAT: No actionable findings in supplied evidence"
	case "lat_unavailable":
		label = "LAT: Review unavailable"
	case "lat_memory_unavailable":
		label = "LAT: Memory compaction unavailable"
	case "lat_discarded":
		label = "LAT: Request changed; review discarded"
	}
	text := f.Language.Text(label)
	if event.Text != "" {
		text += "\n" + f.Redact(event.Text)
	}
	return viewport.Block{Text: text, Markdown: true}
}
