// Purpose: Project conversation messages and Thinking into styled, spaced terminal blocks.
// Pattern: Presentation projection.
// Usage: Session history and live output supply values and a redactor.
// Related: internal/tui/transcript/events.go, internal/agent/model/model.go, internal/tui/viewport/viewport.go.
package transcript

import (
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/tui/locale"
	"github.com/keonho-kim/meca/internal/tui/viewport"
	"strings"
)

type Formatter struct {
	Language locale.Language
	Redact   func(string) string
}

func (f Formatter) Message(m modelapi.Message) []viewport.Block {
	if m.Role == "user" {
		return []viewport.Block{{Text: f.User(f.Redact(m.Content))}}
	}
	if m.Role == "assistant" && m.Content != "" {
		return []viewport.Block{Answer(f.Redact(m.Content))}
	}
	return nil
}

func (f Formatter) User(text string) string { return "> " + text + "\n\n" + f.Language.Text("Agent") }
func Reply(text string) string {
	if strings.TrimSpace(text) == "" {
		return ""
	}
	return strings.TrimRight(text, "\n") + "\n\n"
}

func Answer(text string) viewport.Block { return viewport.Block{Text: Reply(text), Markdown: true} }

// Thought projects model reasoning as dimmed Markdown without changing its source.
func Thought(text string) viewport.Block {
	return viewport.Block{Text: text, Markdown: true, Thinking: true}
}
