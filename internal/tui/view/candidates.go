// Purpose: Render bounded completion candidates below the composer.
// Pattern: Terminal renderer.
// Usage: Chat renders the candidate labels and selected position.
// Related: internal/tui/session/render_hints.go, internal/tui/view/chat.go.
package view

import (
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/meca/internal/tui/locale"
)

const MaxCandidateRows = 7

type Candidates struct {
	Language locale.Language
	Open     bool
	Labels   []string
	Selected int
}

func candidates(screen tcell.Screen, display Candidates, y, width, height int) {
	if !display.Open {
		return
	}
	count := min(MaxCandidateRows, max(0, height-y))
	if count == 0 {
		return
	}
	if len(display.Labels) == 0 {
		Text(screen, 0, y, width, display.Language.Text("No matches"), tcell.StyleDefault.Dim(true))
		return
	}
	start := max(0, display.Selected-count+1)
	for i := start; i < min(len(display.Labels), start+count); i++ {
		Text(screen, 0, y, width, display.Labels[i], SelectionStyle(i == display.Selected))
		y++
	}
}
