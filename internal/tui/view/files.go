// Purpose: Render file-column snapshots and the focused browser composer.
// Pattern: Terminal renderer.
// Usage: Session supplies sanitized labels and presentation positions.
// Related: internal/tui/session/render_files.go, internal/tui/view/input.go.
package view

import (
	"github.com/gdamore/tcell/v3"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/tui/input"
)

type FileColumn struct {
	Title           string
	Labels          []string
	Selected, X     int
	Active, Divider bool
}
type FilesDisplay struct {
	Activity                   Activity
	Usage                      modelapi.TokenUsage
	Columns                    []FileColumn
	Path, Status               string
	ColumnWidth, SelectedCount int
	Input                      input.Window
	InputFocused               bool
	BrowserWidth               int
	Preview                    *FilePreviewDisplay
}

func Files(screen tcell.Screen, display FilesDisplay, w, h int) {
	if display.Preview != nil && display.Preview.Full {
		topLine(screen, display.Path, display.Usage, w, display.Activity)
		FilePreview(screen, *display.Preview, w, h)
		return
	}
	width := display.ColumnWidth
	topLine(screen, display.Path, display.Usage, w, display.Activity)
	for _, column := range display.Columns {
		x := column.X
		Text(screen, x, 2, width, column.Title, tcell.StyleDefault)
		for index := 0; index < len(column.Labels); index++ {
			Text(screen, x, 3+index, width, column.Labels[index], SelectionStyle(column.Active && index == column.Selected))
		}
		if column.Divider {
			for y := 2; y < h-3; y++ {
				Text(screen, x+width, y, 1, "│", tcell.StyleDefault)
			}
		}
	}
	Text(screen, 0, h-3, display.BrowserWidth, display.Activity.Language.Format("selected: %d", display.SelectedCount), tcell.StyleDefault)
	if display.Status != "" {
		Text(screen, 0, h-2, display.BrowserWidth, display.Status, tcell.StyleDefault)
	}
	Input(screen, display.Input, h-1, display.BrowserWidth)
	if display.Preview != nil {
		FilePreview(screen, *display.Preview, w, h)
	}
	if !display.InputFocused {
		screen.HideCursor()
	}
}
