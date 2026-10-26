// Purpose: Represent styled terminal text and wrap it without splitting grapheme clusters.
// Pattern: Pure presentation layout.
// Usage: Viewport and streaming projection consume Line and Rows.
// Related: internal/tui/markdown/render.go, internal/tui/view/chat.go.
package markdown

import (
	"github.com/keonho-kim/meca/internal/tui/textlayout"
	"github.com/rivo/uniseg"
	"iter"
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

// SafeText replaces terminal controls and bidirectional overrides in literal model text.
func SafeText(text string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' && r != '\t' || r >= 0x202a && r <= 0x202e || r >= 0x2066 && r <= 0x2069 {
			return '�'
		}
		return r
	}, text)
}
func line(spans []Span) Line {
	if len(spans) == 1 {
		return Line{Text: spans[0].Text, Spans: spans}
	}
	var text strings.Builder
	size := 0
	for _, s := range spans {
		size += len(s.Text)
	}
	text.Grow(size)
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
func Rows(spans []Span, width int) iter.Seq[Line] {
	return func(yield func(Line) bool) {
		if width < 1 {
			return
		}
		text := line(spans).Text
		ends := make([]int, len(spans))
		end := 0
		for i, span := range spans {
			end += len(span.Text)
			ends[i] = end
		}
		styleIndex := 0
		for wrapped := range textlayout.Rows(text, width) {
			if !strings.ContainsRune(wrapped.Text, '\t') && strings.TrimPrefix(wrapped.Text, wrapped.Prefix) == text[wrapped.Start:wrapped.End] {
				var current []Span
				current = add(current, wrapped.Prefix, 0)
				at := wrapped.Start
				for at < wrapped.End {
					for styleIndex+1 < len(ends) && at >= ends[styleIndex] {
						styleIndex++
					}
					end := min(wrapped.End, ends[styleIndex])
					current = add(current, text[at:end], spans[styleIndex].Style)
					at = end
				}
				if !yield(Line{Text: wrapped.Text, Spans: current, Copy: textlayout.Fragment{Source: text, Row: wrapped}}) {
					return
				}
				continue
			}
			var current []Span
			current = add(current, wrapped.Prefix, 0)
			cells := len(wrapped.Prefix)
			g := uniseg.NewGraphemes(text[wrapped.Start:wrapped.End])
			for g.Next() {
				start, _ := g.Positions()
				for styleIndex+1 < len(ends) && wrapped.Start+start >= ends[styleIndex] {
					styleIndex++
				}
				part, size := g.Str(), g.Width()
				if part == "\t" {
					part, size = "    ", 4
				}
				if size > width-cells {
					part, size = "�", 1
				}
				current = add(current, part, spans[styleIndex].Style)
				cells += size
			}
			row := line(current)
			row.Copy = textlayout.Fragment{Source: text, Row: wrapped}
			if !yield(row) {
				return
			}
		}
	}
}
