// Purpose: Wrap literal text while retaining exact source offsets for terminal hit testing and copying.
// Pattern: Pure layout module.
// Usage: Input, viewport and file preview projections share Wrap and Row.Offset.
// Related: internal/tui/input/layout.go, internal/tui/viewport/viewport.go.
package textlayout

import (
	"github.com/rivo/uniseg"
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
	if width < 1 {
		return nil
	}
	var rows []Row
	base := 0
	for _, paragraph := range strings.Split(source, "\n") {
		indent := paragraph[:len(paragraph)-len(strings.TrimLeft(paragraph, " "))]
		indent = indent[:min(len(indent), max(0, width-2))]
		row := Row{Start: base}
		var line strings.Builder
		cells := 0
		g := uniseg.NewGraphemes(paragraph)
		for g.Next() {
			start, end := g.Positions()
			part := strings.ReplaceAll(g.Str(), "\t", "    ")
			size := uniseg.StringWidth(part)
			if cells+size > width && line.Len() > 0 {
				row.Text, row.End, row.Soft = line.String(), base+start, true
				rows = append(rows, row)
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
		rows = append(rows, row)
		base += len(paragraph) + 1
	}
	return rows
}

// Offset snaps a terminal column to the start or end of the whole grapheme.
func (r Row) Offset(source string, x int, after bool) int {
	column := len(r.Prefix)
	g := uniseg.NewGraphemes(source[r.Start:r.End])
	for g.Next() {
		start, end := g.Positions()
		size := uniseg.StringWidth(strings.ReplaceAll(g.Str(), "\t", "    "))
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
	return len(r.Prefix) + uniseg.StringWidth(strings.ReplaceAll(source[r.Start:min(r.End, max(r.Start, offset))], "\t", "    "))
}
