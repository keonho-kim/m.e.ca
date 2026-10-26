// Purpose: Render bounded completion candidates beneath their input field.
// Pattern: Terminal renderer.
// Usage: Chat and settings render candidate labels at their input column.
// Related: internal/tui/session/render_hints.go, internal/tui/view/chat.go, internal/tui/view/settings.go.
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
	candidatesAt(screen, display, 0, y, width, height)
}

func candidatesAt(screen tcell.Screen, display Candidates, x, y, width, height int) int {
	if !display.Open {
		return 0
	}
	count := min(MaxCandidateRows, max(0, height-y))
	if count == 0 {
		return 0
	}
	if len(display.Labels) == 0 {
		Text(screen, x, y, width, display.Language.Text("No matches"), tcell.StyleDefault.Dim(true))
		return 1
	}
	start := max(0, display.Selected-count+1)
	for i := start; i < min(len(display.Labels), start+count); i++ {
		Text(screen, x, y, width, display.Labels[i], SelectionStyle(i == display.Selected))
		y++
	}
	return max(0, min(len(display.Labels)-start, count))
}
