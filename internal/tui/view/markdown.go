// Purpose: Apply native monochrome attributes to already wrapped Markdown spans.
// Pattern: Terminal rendering adapter.
// Usage: Chat draws styled agent lines without ANSI or HTML.
// Related: internal/tui/markdown/layout.go, internal/tui/view/chat.go.
package view

import (
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/meca/internal/tui/markdown"
	"github.com/rivo/uniseg"
)

func styledText(screen tcell.Screen, y, width int, spans []markdown.Span, base tcell.Style) {
	x := 0
	for _, span := range spans {
		style := base
		if span.Style&markdown.Bold != 0 {
			style = style.Bold(true)
		}
		if span.Style&markdown.Italic != 0 {
			style = style.Italic(true)
		}
		if span.Style&markdown.Underline != 0 {
			style = style.Underline(true)
		}
		if span.Style&markdown.Dim != 0 {
			style = style.Dim(true)
		}
		Text(screen, x, y, width-x, span.Text, style)
		x += uniseg.StringWidth(span.Text)
		if x >= width {
			return
		}
	}
}
