// Purpose: Render styled Markdown or numbered raw files within split or full content bounds.
// Pattern: Terminal renderer.
// Usage: Session supplies already wrapped, redacted rows and the requested horizontal bounds.
// Related: internal/tui/session/render_files.go, internal/tui/browser/preview.go.
package view

import (
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/meca/internal/tui/viewport"
)

type FilePreviewDisplay struct {
	Path, Mode string
	Rows       []viewport.Line
	Full       bool
	X          int
}

func FilePreview(screen tcell.Screen, p FilePreviewDisplay, width, height int) {
	if !p.Full {
		for y := 1; y < height; y++ {
			Text(screen, p.X-1, y, 1, "|", tcell.StyleDefault.Dim(true))
		}
	}
	title := p.Path
	if p.Mode != "" {
		title = p.Mode + " · " + title
	}
	Text(screen, p.X, 1, width-p.X, title, tcell.StyleDefault.Bold(true))
	for i, line := range p.Rows {
		if i+2 >= height {
			break
		}
		if len(line.Spans) > 0 {
			styledText(screen, p.X, i+2, width-p.X, line.Spans, tcell.StyleDefault)
		} else {
			Text(screen, p.X, i+2, width-p.X, line.Text, tcell.StyleDefault)
		}
	}
	screen.HideCursor()
}
