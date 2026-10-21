// Purpose: Render plain file columns with underlined directory names and the focused composer.
// Pattern: Terminal renderer.
// Usage: Session supplies sanitized names, file types, selection state and presentation positions.
// Related: internal/tui/session/render_files.go, internal/tui/view/input.go.
package view

import (
	"github.com/gdamore/tcell/v3"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/tui/input"
)

type FileEntry struct {
	Name              string
	Directory, Marked bool
}

type FileColumn struct {
	Title           string
	Entries         []FileEntry
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
		Text(screen, x, 2, width, column.Title, tcell.StyleDefault.Underline(tcell.UnderlineStyleDotted))
		for index, entry := range column.Entries {
			style := SelectionStyle(column.Active && index == column.Selected)
			prefix := "[ ] "
			if entry.Marked {
				prefix = "[x] "
			}
			Text(screen, x, 3+index, min(4, width), prefix, style)
			name := entry.Name
			if entry.Directory {
				name += "/"
				style = style.Underline(tcell.UnderlineStyleDotted)
			}
			Text(screen, x+4, 3+index, max(0, width-4), name, style)
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
