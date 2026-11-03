// Purpose: Format public peer speech and membership using sparse IRC-style attribution.
// Pattern: Pure presentation projection.
// Usage: History and live team events share these blocks; transport metadata stays hidden.
// Related: internal/tui/transcript/events.go, internal/tui/session/team_events.go.
package transcript

import (
	"strings"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/tui/viewport"
)

func (f Formatter) teamEvent(event runrecord.Event) []viewport.Block {
	if event.Kind == "team_notice" {
		label := "입장했습니다"
		if event.Text == "left" {
			label = "작업을 마치고 나갔습니다"
		}
		return []viewport.Block{{Text: event.Agent + " · " + f.Language.Text(label), Thinking: true}}
	}
	text := f.Redact(event.Text)
	if strings.TrimSpace(text) == "" {
		return nil
	}
	return []viewport.Block{f.TeamHeader(event.Agent, event.Final), {Text: text, Markdown: true}}
}

// TeamHeader shares attribution between incremental streams and committed history.
func (f Formatter) TeamHeader(agent string, final bool) viewport.Block {
	header := "\n\n**" + agent + "**"
	if final {
		header += " · " + f.Language.Text("작업 결과")
	}
	return viewport.Block{Text: header + "\n\n", Markdown: true, KeepWithNext: true}
}
