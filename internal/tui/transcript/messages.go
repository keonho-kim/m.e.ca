// Purpose: Project conversation messages and Plan descriptions into spaced terminal turns.
// Pattern: Presentation projection.
// Usage: Session history and live output supply values and a redactor.
// Related: internal/tui/transcript/events.go, internal/agent/model/model.go, internal/tui/viewport/viewport.go.
package transcript

import (
	"fmt"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	planning "github.com/keonho-kim/v-code/internal/agent/plan"
	"github.com/keonho-kim/v-code/internal/tui/viewport"
	"strings"
)

type Formatter struct{ Redact func(string) string }

func (f Formatter) Message(m modelapi.Message) []viewport.Block {
	if m.Role == "user" {
		return []viewport.Block{{Text: User(f.Redact(m.Content))}}
	}
	if m.Role == "assistant" && m.Planning {
		proposal, err := planning.ParsePlanProposal(m.Content)
		if err == nil {
			value := proposal.Objective
			for index, step := range proposal.Steps {
				value += fmt.Sprintf("\n%d. %s", index+1, step)
			}
			return []viewport.Block{Answer(f.Redact(value))}
		}
	}
	if m.Role == "assistant" && m.Content != "" {
		return []viewport.Block{Answer(f.Redact(m.Content))}
	}
	return nil
}

func User(text string) string { return "> " + text + "\n\nAgent" }
func Reply(text string) string {
	if strings.TrimSpace(text) == "" {
		return ""
	}
	return strings.TrimRight(text, "\n") + "\n\n"
}

func Plan(p *planning.Plan) string {
	if p == nil {
		return ""
	}
	text := fmt.Sprintf("Plan revision %d: %s\n", p.Revision, p.Objective)
	for i, step := range p.Steps {
		text += fmt.Sprintf("%d. %s [%s]\n", i+1, step.Description, step.Status)
	}
	return text + string(p.Status)
}

func Answer(text string) viewport.Block { return viewport.Block{Text: Reply(text), Markdown: true} }
