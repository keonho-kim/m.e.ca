// Purpose: Render a literal numbered file in a right-hand pane or the full content area.
// Pattern: Terminal renderer.
// Usage: Session supplies already wrapped, redacted rows and the requested horizontal bounds.
// Related: internal/tui/session/render_files.go, internal/tui/browser/preview.go.
package view

import "github.com/gdamore/tcell/v3"

type FilePreviewDisplay struct {
	Path, Command string
	Rows          []string
	Full          bool
	X             int
}

func FilePreview(screen tcell.Screen, p FilePreviewDisplay, width, height int) {
	if !p.Full {
		for y := 1; y < height; y++ {
			Text(screen, p.X-1, y, 1, "|", tcell.StyleDefault.Dim(true))
		}
	}
	Text(screen, p.X, 1, width-p.X, p.Path, tcell.StyleDefault.Bold(true))
	end := height
	if p.Command != "" {
		end--
		Text(screen, p.X, end, width-p.X, "> "+p.Command, tcell.StyleDefault)
	}
	for i, line := range p.Rows {
		if i+2 >= end {
			break
		}
		Text(screen, p.X, i+2, width-p.X, line, tcell.StyleDefault)
	}
	screen.HideCursor()
}
