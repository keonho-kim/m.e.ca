// Purpose: Represent styled terminal text and wrap it without splitting grapheme clusters.
// Pattern: Pure presentation layout.
// Usage: Viewport and streaming projection consume Line and Wrap.
// Related: internal/tui/markdown/render.go, internal/tui/view/chat.go.
package markdown

import (
	"github.com/rivo/uniseg"
	"strings"
	"unicode"
)

type Style uint8

const (
	Bold Style = 1 << iota
	Italic
	Underline
	Dim
)

type Span struct {
	Text  string
	Style Style
}
type Line struct {
	Text  string
	Spans []Span
}

func safe(text string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' && r != '\t' || r >= 0x202a && r <= 0x202e || r >= 0x2066 && r <= 0x2069 {
			return '�'
		}
		return r
	}, text)
}
func line(spans []Span) Line {
	var text strings.Builder
	for _, s := range spans {
		text.WriteString(s.Text)
	}
	return Line{Text: text.String(), Spans: spans}
}
func add(spans []Span, text string, style Style) []Span {
	if text == "" {
		return spans
	}
	if len(spans) > 0 && spans[len(spans)-1].Style == style {
		spans[len(spans)-1].Text += text
		return spans
	}
	return append(spans, Span{Text: text, Style: style})
}
func Wrap(spans []Span, width int) []Line {
	if width < 1 {
		return nil
	}
	// Segment the combined text so style boundaries cannot split a joined emoji.
	var body strings.Builder
	var ends []int
	for _, span := range spans {
		body.WriteString(strings.ReplaceAll(span.Text, "\t", "    "))
		ends = append(ends, body.Len())
	}
	text := body.String()
	indent := text[:len(text)-len(strings.TrimLeft(text, " "))]
	indent = indent[:min(len(indent), max(0, width-2))]
	var rows []Line
	var current []Span
	cells, styleIndex := 0, 0
	g := uniseg.NewGraphemes(text)
	for g.Next() {
		start, _ := g.Positions()
		for styleIndex+1 < len(ends) && start >= ends[styleIndex] {
			styleIndex++
		}
		part := g.Str()
		if part == "\n" {
			rows = append(rows, line(current))
			current = nil
			cells = 0
			continue
		}
		size := uniseg.StringWidth(part)
		if cells+size > width && len(current) > 0 {
			rows = append(rows, line(current))
			current = nil
			current = add(current, indent, 0)
			cells = len(indent)
		}
		if size > width {
			part = "�"
			size = 1
		}
		current = add(current, part, spans[styleIndex].Style)
		cells += size
	}
	rows = append(rows, line(current))
	return rows
}
