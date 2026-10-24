// Purpose: Format selected path references above composers within their display budget.
// Pattern: Pure presentation transformation and terminal projection.
// Usage: Chat and file views share ordered, redacted paths without changing editable text.
// Related: internal/tui/view/chat.go, internal/tui/view/files.go, internal/tui/session/render_files.go.
package view

import (
	"strings"

	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/meca/internal/tui/locale"
	"github.com/keonho-kim/meca/internal/tui/textlayout"
	"github.com/rivo/uniseg"
)

const MaxReferenceRows = 3

func ReferenceLines(paths []string, width, rows int, language locale.Language, redact func(string) string) []string {
	rows = min(rows, MaxReferenceRows)
	if len(paths) == 0 || width < 1 || rows < 1 {
		return nil
	}
	text := language.Format("Selected paths (%d): ", len(paths))
	var lines []string
	for index, path := range paths {
		if index > 0 {
			text += " · "
		}
		text += redact(path)
		lines = lines[:0]
		for row := range textlayout.Rows(text, width) {
			lines = append(lines, row.Text)
			if len(lines) > rows {
				break
			}
		}
		if len(lines) > rows {
			break
		}
	}
	if len(lines) > rows {
		lines = lines[:rows]
		clusters := uniseg.NewGraphemes(lines[rows-1])
		var last strings.Builder
		cells := 0
		for clusters.Next() {
			cells += clusters.Width()
			if cells > width-1 {
				break
			}
			last.WriteString(clusters.Str())
		}
		lines[rows-1] = last.String() + "…"
	}
	return lines
}

func drawReferences(screen tcell.Screen, lines []string, y, width, count int) {
	for i, line := range lines[:min(len(lines), count)] {
		Text(screen, 0, y+i, width, line, tcell.StyleDefault.Dim(true))
	}
}
