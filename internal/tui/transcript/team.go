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
			label = "퇴장했습니다"
		}
		return []viewport.Block{{Text: event.Agent + " · " + f.Language.Text(label), Thinking: true}}
	}
	text := event.Text
	if event.Team != nil && event.Team.Work != nil {
		work := event.Team.Work
		if len(work.Paths) > 0 {
			text += "\n\n" + f.Language.Text("작업 경로") + ": " + strings.Join(work.Paths, ", ")
		}
		if work.Contract != "" {
			text += "\n\n" + f.Language.Text("함께 맞출 내용") + ": " + work.Contract
		}
	}
	body := f.TeamBody(text)
	if body.Text == "" {
		return nil
	}
	header := f.TeamHeader(event.Agent, event.Final)
	if event.Team != nil {
		target := event.Team.ReplyAuthor
		if target == "user" {
			target = f.Language.Text("사용자")
		}
		for _, name := range event.Team.Mentions {
			if name == event.Team.ReplyAuthor {
				continue
			}
			if target != "" {
				target += ", "
			}
			target += name
		}
		if target != "" {
			header.Text += f.Language.Format(" · %s에게", f.Redact(target))
		}
	}
	return []viewport.Block{header, body}
}

// TeamBody shares source-preserving redaction and spacing across live and saved
// public speech without allocating an unused attribution block for live updates.
func (f Formatter) TeamBody(text string) viewport.Block {
	text = f.Redact(text)
	if strings.TrimSpace(text) == "" {
		return viewport.Block{}
	}
	return viewport.Block{Text: text, Markdown: true, SpaceAfter: 2}
}

// TeamHeader shares attribution between incremental streams and committed history.
func (f Formatter) TeamHeader(agent string, final bool) viewport.Block {
	header := "**" + agent + "**"
	if final {
		header += " · " + f.Language.Text("작업 결과")
	}
	return viewport.Block{Text: header, Markdown: true, KeepWithNext: true, SpaceAfter: 1}
}
