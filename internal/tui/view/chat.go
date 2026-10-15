// Purpose: Render projected transcript lines and the flowing composer without changing viewport state.
// Pattern: Terminal renderer.
// Usage: Session supplies ChatDisplay after calculating its read position.
// Related: internal/tui/session/render_chat.go, internal/tui/view/candidates.go, internal/tui/view/header.go.
package view

import (
	"github.com/gdamore/tcell/v3"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/tui/input"
	"github.com/keonho-kim/v-code/internal/tui/viewport"
)

type ChatDisplay struct {
	Usage            modelapi.TokenUsage
	Lines            []viewport.Line
	Start, Available int
	Editor           input.Editor
	InputMaxLines    int
	Candidates       Candidates
}

func Chat(screen tcell.Screen, display ChatDisplay, width, height int) {
	topLine(screen, "", display.Usage, width)
	y := 1
	for i := display.Start; i < len(display.Lines) && y-1 < display.Available; i++ {
		style := tcell.StyleDefault
		if display.Lines[i].Thinking {
			style = style.Dim(true)
		}
		if len(display.Lines[i].Spans) > 0 {
			styledText(screen, y, width, display.Lines[i].Spans, style)
		} else {
			Text(screen, 0, y, width, display.Lines[i].Text, style)
		}
		y++
	}
	if y > 1 && display.Lines[display.Start+y-2].Text != "" {
		y++
	}
	y += Input(screen, display.Editor, y, width, min(display.InputMaxLines, height-y))
	candidates(screen, display.Candidates, y, width, height)
}
