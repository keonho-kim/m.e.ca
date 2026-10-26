// Purpose: Wrap literal text while retaining exact source offsets for terminal hit testing and copying.
// Pattern: Pure layout module.
// Usage: Input, viewport and file preview projections share Wrap and Row.Offset.
// Related: internal/tui/input/layout.go, internal/tui/viewport/viewport.go.
package textlayout

import (
	"github.com/rivo/uniseg"
	"iter"
	"slices"
	"strings"
)

type Row struct {
	Text       string
	Start, End int
	Prefix     string
	Soft       bool
}

// Fragment maps one displayed row to its original, unwrapped text.
type Fragment struct {
	Source string
	Row    Row
}

func Wrap(source string, width int) []Row {
	return slices.Collect(Rows(source, width))
}

// Rows permits viewport consumers to stop without allocating unseen rows.
func Rows(source string, width int) iter.Seq[Row] {
	return RowsFrom(source, width, 0, "")
}

// RowsFrom resumes at a previously indexed row without copying the suffix.
// prefix is that row's continuation indentation, not part of source offsets.
func RowsFrom(source string, width, start int, prefix string) iter.Seq[Row] {
	return func(yield func(Row) bool) {
		if width < 1 || start < 0 || start > len(source) {
			return
		}
		base := start
		for {
			paragraph, rest, more := strings.Cut(source[base:], "\n")
			if !strings.ContainsRune(paragraph, '\t') && len(prefix)+Width(paragraph) <= width {
				if !yield(Row{Text: prefix + paragraph, Prefix: prefix, Start: base, End: base + len(paragraph)}) {
					return
				}
				if !more {
					return
				}
				base = len(source) - len(rest)
				prefix = ""
				continue
			}
			indent := paragraph[:len(paragraph)-len(strings.TrimLeft(paragraph, " "))]
			if base == start && start > 0 && source[start-1] != '\n' {
				indent = prefix
			}
			indent = indent[:min(len(indent), max(0, width-2))]
			row := Row{Start: base, Prefix: prefix}
			var line strings.Builder
			line.WriteString(prefix)
			cells := len(prefix)
			g := uniseg.NewGraphemes(paragraph)
			for g.Next() {
				start, end := g.Positions()
				part, size := g.Str(), g.Width()
				if part == "\t" {
					part, size = "    ", 4
				}
				if cells+size > width && line.Len() > 0 {
					row.Text, row.End, row.Soft = line.String(), base+start, true
					if !yield(row) {
						return
					}
					row = Row{Start: base + start, Prefix: indent}
					line.Reset()
					line.WriteString(indent)
					cells = len(indent)
				}
				if size > width-cells {
					part, size = "�", 1
				}
				line.WriteString(part)
				cells += size
				row.End = base + end
			}
			row.Text, row.End = line.String(), base+len(paragraph)
			if !yield(row) {
				return
			}
			if !more {
				return
			}
			base = len(source) - len(rest)
			prefix = ""
		}
	}
}

// Offset snaps a terminal column to the start or end of the whole grapheme.
func (r Row) Offset(source string, x int, after bool) int {
	column := len(r.Prefix)
	g := uniseg.NewGraphemes(source[r.Start:r.End])
	for g.Next() {
		start, end := g.Positions()
		size := g.Width()
		if g.Str() == "\t" {
			size = 4
		}
		if x < column+size {
			if after && x >= column {
				return r.Start + end
			}
			return r.Start + start
		}
		column += size
	}
	return r.End
}

func (r Row) Column(source string, offset int) int {
	return len(r.Prefix) + Width(strings.ReplaceAll(source[r.Start:min(r.End, max(r.Start, offset))], "\t", "    "))
}
