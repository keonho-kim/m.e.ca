// Purpose: Render menu snapshots with inline model identity, connection fields and previews.
// Pattern: Terminal renderer.
// Usage: Session supplies MenuDisplay without transferring mutable session state.
// Related: internal/tui/session/render_menu.go, internal/tui/view/preview.go, internal/tui/view/field.go.
package view

import (
	"github.com/gdamore/tcell/v3"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/tui/input"
	"github.com/rivo/uniseg"
	"strings"
)

type MenuDisplay struct {
	Usage                  modelapi.TokenUsage
	Model, IdentifiedModel string
	Title, Status, Command string
	Field                  *input.Editor
	Secret                 bool
	Preview                []PreviewLine
	Plan, Choices, Details []string
	Selected               int
	Thinking               *bool
	ThinkingSupported      bool
	ThinkingOnOnly         bool
}

func Menu(screen tcell.Screen, display MenuDisplay, w, h int) {
	topLine(screen, display.Title, display.Usage, w)
	if display.Model != "" {
		Text(screen, 0, 1, w, display.Model, tcell.StyleDefault)
		x := uniseg.StringWidth(display.Model) + 2
		if x < w {
			Text(screen, x, 1, w-x, display.IdentifiedModel, tcell.StyleDefault.Dim(true))
		}
	}
	if display.Details != nil {
		for i, line := range display.Details {
			if i+2 >= h {
				break
			}
			Text(screen, 0, i+2, w, line, tcell.StyleDefault)
		}
		return
	}
	if display.Field != nil {
		Field(screen, *display.Field, 2, w, display.Secret)
		if display.Status != "" {
			Text(screen, 0, min(4, h-1), w, display.Status, tcell.StyleDefault)
		}
		return
	}
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
	for _, line := range display.Plan {
		if y >= h-5 {
			break
		}
		Text(screen, 0, y, w, line, tcell.StyleDefault)
		y++
	}
	visible := max(1, h-y-2)
	start := max(0, display.Selected-visible+1)
	for i := start; i < min(len(display.Choices), start+visible); i++ {
		if display.Thinking != nil && i == 1 {
			if display.ThinkingOnOnly {
				Text(screen, 0, y, w, "Thinking ", SelectionStyle(i == display.Selected))
				Text(screen, 9, y, max(0, w-9), "ON", tcell.StyleDefault.Bold(true))
				Text(screen, 11, y, max(0, w-11), " (OFF unavailable)", tcell.StyleDefault.Dim(true))
			} else {
				Thinking(screen, y, w, *display.Thinking, display.ThinkingSupported, i == display.Selected)
			}
		} else {
			Text(screen, 0, y, w, display.Choices[i], SelectionStyle(i == display.Selected))
		}
		y++
	}
	if display.Status != "" && y < h {
		Text(screen, 0, y, w, display.Status, tcell.StyleDefault)
	}
	if strings.HasPrefix(display.Command, "!") {
		Text(screen, 0, h-1, w, "> "+display.Command, tcell.StyleDefault)
	}
}
