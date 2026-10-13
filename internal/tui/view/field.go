// Purpose: Render a single-line connection field with its real cursor and a bounded horizontal window.
// Pattern: Presentation renderer.
// Usage: Endpoint, key and model ID screens supply their editor snapshot; secret values remain masked.
// Related: internal/tui/session/render.go, internal/tui/input/editor.go, internal/tui/view/cells.go.
package view

import (
	"strings"

	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/v-code/internal/tui/input"
	"github.com/rivo/uniseg"
)

func Field(screen tcell.Screen, editor input.Editor, y, width int, masked bool) {
	value := editor.String()
	before := string(editor.Text[:editor.Cursor])
	if masked {
		value = strings.Repeat("*", len(editor.Text))
		before = strings.Repeat("*", editor.Cursor)
	}
	cursor := uniseg.StringWidth(before)
	offset := max(0, cursor-max(1, width-2)+1)
	start, cells := len(value), 0
	g := uniseg.NewGraphemes(value)
	for g.Next() {
		from, _ := g.Positions()
		if cells >= offset {
			start = from
			break
		}
		cells += uniseg.StringWidth(g.Str())
	}
	Text(screen, 0, y, width, "> "+value[start:], tcell.StyleDefault)
	screen.ShowCursor(min(width-1, 2+max(0, cursor-cells)), y)
}
