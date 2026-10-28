// Purpose: Render a spaced transcript and bottom-anchored composer without changing viewport state.
// Pattern: Terminal renderer.
// Usage: Session supplies ChatDisplay after calculating its read position.
// Related: internal/tui/session/render_chat.go, internal/tui/view/candidates.go, internal/tui/view/header.go.
package view

import (
	"github.com/gdamore/tcell/v3"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/tui/input"
	"github.com/keonho-kim/meca/internal/tui/viewport"
	"strings"
)

type ChatDisplay struct {
	Notice     string
	Activity   Activity
	Usage      modelapi.TokenUsage
	Lines      []viewport.Line
	FocusedID  int
	Scrollbar  viewport.Scrollbar
	Input      input.Window
	References []string
	Layout     ComposerLayout
	Candidates Candidates
}

// TranscriptWidth reserves a scrollbar cell and five blank cells before it.
// Keeping this width before overflow avoids rewrapping when the scrollbar appears.
func TranscriptWidth(width int) int { return max(1, width-1-5) }

func Chat(screen tcell.Screen, display ChatDisplay, width int) {
	topLine(screen, "", display.Usage, width, display.Activity)
	contentWidth := TranscriptWidth(width)
	y := 1
	for i := 0; i < len(display.Lines) && y-1 < display.Layout.TranscriptRows; i++ {
		style := tcell.StyleDefault
		if display.Lines[i].Thinking {
			style = style.Dim(true)
		}
		if display.FocusedID != 0 && display.Lines[i].ID == display.FocusedID {
			style = tcell.StyleDefault.Foreground(tcell.NewHexColor(0x7aa2c1)).Bold(true)
		}
		if display.Lines[i].Change != 0 {
			style = ChangeStyle(display.Lines[i].Change)
			// Keep the tool indentation and the scrollbar gutter outside the diff.
			Text(screen, 4, y, contentWidth-4, strings.Repeat(" ", max(0, contentWidth-4)), style)
			Text(screen, 4, y, contentWidth-4, strings.TrimPrefix(display.Lines[i].Text, "    "), style)
		} else if len(display.Lines[i].Spans) > 0 {
			styledText(screen, 0, y, contentWidth, display.Lines[i].Spans, style)
		} else {
			Text(screen, 0, y, contentWidth, display.Lines[i].Text, style)
		}
		y++
	}
	Scrollbar(screen, width-1, 1, display.Scrollbar)
	layout := display.Layout
	candidates(screen, display.Candidates, layout.CandidateY, width, layout.CandidateEnd)
	drawReferences(screen, display.References, layout.ReferenceY, width, layout.ReferenceRows)
	if display.Notice != "" && layout.InputY > 1 {
		Text(screen, 0, layout.InputY-1, width, display.Notice, tcell.StyleDefault.Dim(true))
	}
	Input(screen, display.Input, layout.InputY, width)
}
