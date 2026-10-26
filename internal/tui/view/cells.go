// Purpose: Draw bounded grapheme clusters using their actual terminal cell widths.
// Pattern: Presentation primitive.
// Usage: Used by session and startup renderers for Unicode-safe text.
// Related: internal/tui/view/input.go, internal/tui/view/header.go, internal/tui/startup/resume.go.
package view

import (
	"github.com/gdamore/tcell/v3"
	"github.com/rivo/uniseg"
)

func Text(screen tcell.Screen, x, y, width int, value string, style tcell.Style) {
	g := uniseg.NewGraphemes(value)
	end := x + width
	for g.Next() {
		cells := g.Width()
		if x+cells > end {
			break
		}
		screen.Put(x, y, g.Str(), style)
		x += cells
	}
}
