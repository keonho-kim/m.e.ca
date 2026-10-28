// Purpose: Apply native monochrome attributes to already wrapped Markdown spans.
// Pattern: Terminal rendering adapter.
// Usage: Chat and split/full file viewers draw styled lines without ANSI or HTML.
// Related: internal/tui/markdown/layout.go, internal/tui/view/chat.go, internal/tui/view/file_preview.go.
package view

import (
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/meca/internal/tui/markdown"
	"github.com/rivo/uniseg"
)

func styledText(screen tcell.Screen, x, y, width int, spans []markdown.Span, base tcell.Style) {
	end := x + width
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
		Text(screen, x, y, end-x, span.Text, style)
		x += uniseg.StringWidth(span.Text)
		if x >= end {
			return
		}
	}
}
