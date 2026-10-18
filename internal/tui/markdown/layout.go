// Purpose: Represent styled terminal text and wrap it without splitting grapheme clusters.
// Pattern: Pure presentation layout.
// Usage: Viewport and streaming projection consume Line and Wrap.
// Related: internal/tui/markdown/render.go, internal/tui/view/chat.go.
package markdown

import (
	"github.com/keonho-kim/v-code/internal/tui/textlayout"
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
	Copy  textlayout.Fragment
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
	var body strings.Builder
	var ends []int
	for _, span := range spans {
		body.WriteString(span.Text)
		ends = append(ends, body.Len())
	}
	text := body.String()
	var rows []Line
	styleIndex := 0
	for _, wrapped := range textlayout.Wrap(text, width) {
		var current []Span
		current = add(current, wrapped.Prefix, 0)
		cells := len(wrapped.Prefix)
		g := uniseg.NewGraphemes(text[wrapped.Start:wrapped.End])
		for g.Next() {
			start, _ := g.Positions()
			for styleIndex+1 < len(ends) && wrapped.Start+start >= ends[styleIndex] {
				styleIndex++
			}
			part := strings.ReplaceAll(g.Str(), "\t", "    ")
			size := uniseg.StringWidth(part)
			if size > width-cells {
				part, size = "�", 1
			}
			current = add(current, part, spans[styleIndex].Style)
			cells += size
		}
		row := line(current)
		row.Copy = textlayout.Fragment{Source: text, Row: wrapped}
		rows = append(rows, row)
	}
	return rows
}
