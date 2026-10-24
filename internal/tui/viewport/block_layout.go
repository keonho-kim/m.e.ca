// Purpose: Index wrapped block positions sparsely and materialize only requested terminal rows.
// Pattern: Event-loop-owned presentation index.
// Usage: Transcript queries Count and Window; block replacement releases its index.
// Related: internal/tui/viewport/window.go, internal/tui/markdown/stream.go, internal/tui/textlayout/wrap.go.
package viewport

import (
	"sort"
	"strings"

	"github.com/keonho-kim/meca/internal/tui/markdown"
	"github.com/keonho-kim/meca/internal/tui/textlayout"
)

const maxBoundaryBlanks = 2

type blockPoint struct {
	source, row int
	prefix      string
}
type blockLayout struct {
	key                            blockKey
	ready                          bool
	points                         []blockPoint
	parsed                         []markdown.Line
	metrics                        []rowMetrics
	source                         string
	markdown                       bool
	total, trim, leading, trailing int
}

type rowMetrics struct{ count, leading, trailing int }

func (b *Block) prepare(width int) {
	key := blockKey{b.Text, width, b.ID, b.Markdown, b.Thinking, b.Centered, b.Changes}
	l := &b.layout
	if l.ready && l.key == key {
		return
	}
	previous := *l
	*l = blockLayout{key: key, ready: true}
	l.source, l.markdown = b.Text, b.Markdown && len(b.Text) <= markdown.MaxParseBytes
	if b.Markdown && !l.markdown {
		l.source = markdown.SafeText(b.Text)
	}
	if width < 1 || strings.TrimSpace(b.Text) == "" {
		b.stream = markdown.Stream{}
		return
	}
	if l.markdown {
		l.parsed = b.stream.Render(b.Text)
		l.metrics = previous.metrics[:0]
	}
	lastContent, firstContent := -1, -1
	observe := func(text string) {
		if strings.TrimSpace(text) != "" {
			if firstContent < 0 {
				firstContent = l.total
			}
			lastContent = l.total
		}
		l.total++
	}
	if l.markdown {
		for i, line := range l.parsed {
			if i%256 == 0 {
				l.points = append(l.points, blockPoint{source: i, row: l.total})
			}
			var metric rowMetrics
			if previous.key.width == width && i < b.stream.ReusedRows() && i < len(previous.metrics) {
				metric = previous.metrics[i]
			} else {
				for row := range textlayout.Rows(line.Text, width) {
					if strings.TrimSpace(row.Text) == "" {
						if metric.leading == metric.count {
							metric.leading++
						}
						metric.trailing++
					} else {
						metric.trailing = 0
					}
					metric.count++
				}
			}
			l.metrics = append(l.metrics, metric)
			if metric.leading < metric.count {
				if firstContent < 0 {
					firstContent = l.total + metric.leading
				}
				lastContent = l.total + metric.count - metric.trailing - 1
			}
			l.total += metric.count
		}
	} else {
		b.stream = markdown.Stream{}
		for row := range textlayout.Rows(l.source, width) {
			if l.total%256 == 0 {
				l.points = append(l.points, blockPoint{source: row.Start, row: l.total, prefix: row.Prefix})
			}
			observe(row.Text)
		}
	}
	if b.Markdown || b.Thinking {
		l.trim = max(0, firstContent)
		l.total = min(l.total, lastContent+1+maxBoundaryBlanks)
	}
	l.leading = max(0, firstContent-l.trim)
	l.trailing = l.total - lastContent - 1
}

func (b *Block) Count(width int) int { b.prepare(width); return b.layout.total - b.layout.trim }

func (b *Block) Window(width, start, count int) []Line {
	start = max(0, start)
	if count < 1 || start >= b.Count(width) {
		return nil
	}
	l := &b.layout
	if l.markdown && l.parsed == nil {
		l.parsed = b.stream.Render(b.Text)
	}
	start = max(0, start) + l.trim
	end := min(l.total, start+count)
	at := sort.Search(len(l.points), func(i int) bool { return l.points[i].row > start }) - 1
	point := l.points[max(0, at)]
	row := point.row
	out := make([]Line, 0, end-start)
	appendRow := func(value Line) bool {
		if row >= start {
			value.ID, value.Row, value.Thinking = b.ID, row, b.Thinking
			out = append(out, value)
		}
		row++
		return row < end
	}
	if l.markdown {
		for i := point.source; i < len(l.parsed) && row < end; i++ {
			for wrapped := range markdown.Rows(l.parsed[i].Spans, width) {
				if !appendRow(Line{Text: wrapped.Text, Spans: wrapped.Spans, Copy: wrapped.Copy}) {
					break
				}
			}
		}
	} else {
		for wrapped := range textlayout.RowsFrom(l.source, width, point.source, point.prefix) {
			if !appendRow(Line{Text: wrapped.Text, Copy: textlayout.Fragment{Source: l.source, Row: wrapped}}) {
				break
			}
		}
		b.classifyChanges(out)
	}
	if b.Centered {
		centerLines(out, width)
	}
	return out
}

// ReleaseDisplay retains scroll indexes but drops inactive parsing resources.
func (v *View) ReleaseDisplay() {
	for i := range v.Blocks {
		v.Blocks[i].visible = false
		v.Blocks[i].releaseParsed()
	}
}

func (b *Block) releaseParsed() {
	b.layout.parsed = nil
	b.layout.metrics = nil
	b.stream = markdown.Stream{}
}
