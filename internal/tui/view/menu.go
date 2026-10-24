// Purpose: Render menu snapshots with inline model identity, connection fields and previews.
// Pattern: Terminal renderer.
// Usage: Session supplies MenuDisplay without transferring mutable session state.
// Related: internal/tui/session/render_menu.go, internal/tui/view/preview.go, internal/tui/view/input.go.
package view

import (
	"github.com/gdamore/tcell/v3"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"strings"
)

type MenuDisplay struct {
	Activity               Activity
	Usage                  modelapi.TokenUsage
	Title, Status, Command string
	Preview                []PreviewLine
	Choices                []string
	Selected               int
}

func Menu(screen tcell.Screen, display MenuDisplay, w, h int) {
	topLine(screen, display.Title, display.Usage, w, display.Activity)
	y := 2
	for _, line := range display.Preview {
		if y >= h-5 {
			break
		}
		style := tcell.StyleDefault
		if line.Change != 0 {
			style = ChangeStyle(line.Change)
			Text(screen, 0, y, w, strings.Repeat(" ", w), style)
		}
		Text(screen, 0, y, w, line.Text, style)
		y++
	}
	visible := max(1, h-y-2)
	start := max(0, display.Selected-visible+1)
	for i := start; i < min(len(display.Choices), start+visible); i++ {
		Text(screen, 0, y, w, display.Choices[i], SelectionStyle(i == display.Selected))
		y++
	}
	if display.Status != "" && y < h {
		Text(screen, 0, y, w, display.Status, tcell.StyleDefault)
	}
	if strings.HasPrefix(display.Command, "!") || strings.HasPrefix(display.Command, ":") {
		Text(screen, 0, h-1, w, "> "+display.Command, tcell.StyleDefault)
	}
}
