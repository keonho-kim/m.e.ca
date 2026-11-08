// Purpose: Represent styled terminal text and wrap it without splitting grapheme clusters.
// Pattern: Pure presentation layout.
// Usage: Viewport and streaming projection consume Line and Rows.
// Related: internal/tui/markdown/render.go, internal/tui/view/chat.go.
package markdown

import (
	"iter"
	"strings"
	"unicode"
	"unsafe"

	"github.com/keonho-kim/meca/internal/tui/textlayout"
	"github.com/rivo/uniseg"
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

// RetainedBytes conservatively counts a line's derived text and span storage.
func (line Line) RetainedBytes() int {
	weight := len(line.Text) + cap(line.Spans)*int(unsafe.Sizeof(Span{}))
	for _, span := range line.Spans {
		weight += len(span.Text)
	}
	return weight
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
	return rowsFrom(line(spans).Text, spans, width, 0, "")
}

// RowsFrom resumes one already rendered line at a sparse terminal-row boundary.
func (line Line) RowsFrom(width, start int, prefix string) iter.Seq[Line] {
	return rowsFrom(line.Text, line.Spans, width, start, prefix)
}

func rowsFrom(text string, spans []Span, width, start int, prefix string) iter.Seq[Line] {
	cursor := spanCursor{spans: spans}
	if len(spans) > 0 {
		cursor.end = len(spans[0].Text)
	}
	return styledRows(text, width, start, prefix, cursor)
}

func styledRows(text string, width, start int, prefix string, initial spanCursor) iter.Seq[Line] {
	return func(yield func(Line) bool) {
		if width < 1 {
			return
		}
		cursor := initial
		for wrapped := range textlayout.RowsFrom(text, width, start, prefix) {
			if !strings.ContainsRune(wrapped.Text, '\t') && strings.TrimPrefix(wrapped.Text, wrapped.Prefix) == text[wrapped.Start:wrapped.End] {
				var current []Span
				current = add(current, wrapped.Prefix, 0)
				at := wrapped.Start
				for at < wrapped.End {
					style, until := cursor.at(at)
					end := min(wrapped.End, until)
					current = add(current, text[at:end], style)
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
				style, _ := cursor.at(wrapped.Start + start)
				part, size := g.Str(), g.Width()
				if part == "\t" {
					part, size = "    ", 4
				}
				if size > width-cells {
					part, size = "�", 1
				}
				current = add(current, part, style)
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
