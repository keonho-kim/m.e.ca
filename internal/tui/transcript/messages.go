// Purpose: Project conversation messages into spaced terminal turns.
// Pattern: Presentation projection.
// Usage: Session history and live output supply values and a redactor.
// Related: internal/tui/transcript/events.go, internal/agent/model/model.go, internal/tui/viewport/viewport.go.
package transcript

import (
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/tui/viewport"
	"strings"
)

type Formatter struct{ Redact func(string) string }

func (f Formatter) Message(m modelapi.Message) []viewport.Block {
	if m.Role == "user" {
		return []viewport.Block{{Text: User(f.Redact(m.Content))}}
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

func Answer(text string) viewport.Block { return viewport.Block{Text: Reply(text), Markdown: true} }
