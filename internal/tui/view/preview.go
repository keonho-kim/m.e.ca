// Purpose: Index approval and skill previews sparsely and materialize only visible rows.
// Pattern: Immutable-source layout with bounded viewport projection.
// Usage: Workers build Preview; terminal renderers request Window after Count.
// Related: internal/tui/textlayout/wrap.go, internal/tui/session/skills.go, internal/tui/session/render_menu.go.
package view

import (
	"strings"

	"context"
	"sort"

	"github.com/keonho-kim/meca/internal/tui/textlayout"
)

type PreviewLine struct {
	Text   string
	Change byte
}

type previewPoint struct {
	row, offset int
	prefix      string
	change      byte
	hunk        bool
}
type Preview struct {
	source       string
	width, total int
	diff         bool
	index        []previewPoint
}

func BuildPreview(ctx context.Context, source string, width int, diff bool) (*Preview, error) {
	p := &Preview{source: source, width: max(1, width), diff: diff}
	hunk := false
	var change byte
	for row := range textlayout.Rows(source, p.width) {
		if p.total%256 == 0 {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
		}
		if row.Start == 0 || source[row.Start-1] == '\n' {
			hunk, change = previewChange(source[row.Start:], diff, hunk)
		}
		if p.total%256 == 0 {
			p.index = append(p.index, previewPoint{p.total, row.Start, row.Prefix, change, hunk})
		}
		p.total++
	}
	return p, ctx.Err()
}

func previewChange(line string, diff, hunk bool) (bool, byte) {
	if diff && strings.HasPrefix(line, "@@ -") {
		hunk = true
	}
	if hunk && len(line) > 0 && (line[0] == '+' || line[0] == '-') {
		return hunk, line[0]
	}
	return hunk, 0
}

func (p *Preview) Count() int {
	if p == nil {
		return 0
	}
	return p.total
}

func (p *Preview) Window(start, count int) []PreviewLine {
	if p == nil || count <= 0 || start >= p.total {
		return nil
	}
	start = max(0, start)
	at := sort.Search(len(p.index), func(i int) bool { return p.index[i].row > start }) - 1
	point := p.index[max(0, at)]
	rows := make([]PreviewLine, 0, min(count, p.total-start))
	index, hunk, change := point.row, point.hunk, point.change
	for row := range textlayout.RowsFrom(p.source, p.width, point.offset, point.prefix) {
		if row.Start == 0 || p.source[row.Start-1] == '\n' {
			hunk, change = previewChange(p.source[row.Start:], p.diff, hunk)
		}
		if index >= start {
			rows = append(rows, PreviewLine{row.Text, change})
			if len(rows) == count {
				break
			}
		}
		index++
	}
	return rows
}
