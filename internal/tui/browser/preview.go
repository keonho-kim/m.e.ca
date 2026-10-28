// Purpose: Project previews through cached Markdown or sparse numbered raw text windows.
// Pattern: Pure presentation state with a bounded display window.
// Usage: File preview uses Count for scrolling and Window for its current pane.
// Related: internal/tui/browser/markdown_preview.go, internal/tui/session/render_files.go, internal/tui/textlayout/wrap.go.
package browser

import (
	"fmt"
	"github.com/keonho-kim/meca/internal/tui/textlayout"
	"github.com/keonho-kim/meca/internal/tui/viewport"
	"sort"
	"strconv"
	"strings"
)

const previewStride = 256
const previewTruncated = "[미리보기는 처음 16MiB까지만 표시]"

type previewPoint struct{ offset, line, row int }
type Preview struct {
	formatted                viewport.Block
	index                    []previewPoint
	width, total, digits     int
	content                  string
	Path, Content, Error     string
	Full, Loading, Truncated bool
	Raw                      bool
	Offset                   int
}

func (p *Preview) special(width int) []string {
	if p.Loading {
		return []string{"파일 읽는 중…"}
	}
	if p.Error != "" {
		return viewport.Wrap(p.Error, max(1, width))
	}
	if p.Content == "" {
		p.index, p.content = nil, ""
		p.formatted = viewport.Block{}
		return []string{"(빈 파일)"}
	}
	return nil
}
func (p *Preview) prepare(width int) {
	width = max(1, width)
	if p.width == width && p.content == p.Content && p.index != nil {
		return
	}
	p.width, p.content, p.total = width, p.Content, 0
	text := strings.TrimSuffix(p.Content, "\n")
	count := strings.Count(text, "\n") + 1
	p.digits = max(3, len(strconv.Itoa(count)))
	p.index = make([]previewPoint, 0, (count+previewStride-1)/previewStride)
	offset, line := 0, 0
	for {
		value, rest, more := strings.Cut(text[offset:], "\n")
		if line%previewStride == 0 {
			p.index = append(p.index, previewPoint{offset, line, p.total})
		}
		for range textlayout.Rows(p.sourceLine(value, line), p.bodyWidth()) {
			p.total++
		}
		if !more {
			break
		}
		offset = len(text) - len(rest)
		line++
	}
}
func (p *Preview) bodyWidth() int {
	if p.width <= p.digits+3 {
		return p.width
	}
	return p.width - p.digits - 3
}
func (p *Preview) sourceLine(text string, line int) string {
	text = strings.TrimSuffix(text, "\r")
	if p.width <= p.digits+3 {
		return fmt.Sprintf("%*d | %s", p.digits, line+1, text)
	}
	return text
}
func (p *Preview) Count(width int) int {
	if rows := p.special(width); rows != nil {
		return len(rows)
	}
	if p.Rendered() {
		return p.markdownCount(width)
	}
	p.formatted = viewport.Block{}
	p.prepare(width)
	if p.Truncated {
		return p.total + 1
	}
	return p.total
}
func (p *Preview) Window(width, start, count int) []viewport.Line {
	if count < 1 {
		return nil
	}
	start = max(0, start)
	if start >= p.Count(width) {
		return nil
	}
	if rows := p.special(width); rows != nil {
		var out []viewport.Line
		for _, text := range rows[min(start, len(rows)):min(start+count, len(rows))] {
			out = append(out, viewport.Line{Text: text})
		}
		return out
	}
	if p.Rendered() {
		return p.markdownWindow(width, start, count)
	}
	p.prepare(width)
	rows := make([]viewport.Line, 0, count)
	if start < p.total {
		at := sort.Search(len(p.index), func(i int) bool { return p.index[i].row > start }) - 1
		point := p.index[max(0, at)]
		text := strings.TrimSuffix(p.content, "\n")
		offset, line, row := point.offset, point.line, point.row
		for len(rows) < count {
			value, rest, more := strings.Cut(text[offset:], "\n")
			part := 0
			for wrapped := range textlayout.Rows(p.sourceLine(value, line), p.bodyWidth()) {
				if row >= start {
					prefix := ""
					if p.width > p.digits+3 {
						prefix = strings.Repeat(" ", p.digits+3)
						if part == 0 {
							prefix = fmt.Sprintf("%*d | ", p.digits, line+1)
						}
					}
					rows = append(rows, viewport.Line{Text: prefix + wrapped.Text})
					if len(rows) == count {
						break
					}
				}
				row++
				part++
			}
			if !more {
				break
			}
			offset = len(text) - len(rest)
			line++
		}
	}
	if p.Truncated && len(rows) < count && start+len(rows) >= p.total {
		rows = append(rows, viewport.Line{Text: previewTruncated})
	}
	return rows
}
