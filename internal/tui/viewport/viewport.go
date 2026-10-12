// Purpose: Keep bounded output blocks and preserve an event-relative reading anchor on resize.
// Pattern: Simple Module (presentation projection).
// Usage: Mutated only by the terminal event loop and rendered at the current width.
// Related: internal/tui/session/render.go, internal/tui/session/tasks.go.
package viewport

import (
	"strings"

	"github.com/rivo/uniseg"
)

type Block struct {
	ID       int
	Text     string
	Thinking bool
}
type Line struct {
	ID, Row  int
	Text     string
	Thinking bool
}
type View struct {
	Blocks              []Block
	Next                int
	Follow              bool
	AnchorID, AnchorRow int
}

func (v *View) Append(text string, thinking bool) {
	if text == "" {
		return
	}
	v.Next++
	v.Blocks = append(v.Blocks, Block{ID: v.Next, Text: text, Thinking: thinking})
	v.bound(false)
}
func Wrap(text string, width int) []string {
	if width < 1 {
		return nil
	}
	var out []string
	for _, paragraph := range strings.Split(strings.ReplaceAll(text, "\t", "    "), "\n") {
		line := ""
		cells := 0
		g := uniseg.NewGraphemes(paragraph)
		for g.Next() {
			part := g.Str()
			n := uniseg.StringWidth(part)
			if cells+n > width && line != "" {
				out = append(out, line)
				line = ""
				cells = 0
			}
			line += part
			cells += n
		}
		out = append(out, line)
	}
	return out
}
func (v *View) Lines(width int) []Line {
	var lines []Line
	for _, block := range v.Blocks {
		for row, text := range Wrap(block.Text, width) {
			lines = append(lines, Line{ID: block.ID, Row: row, Text: text, Thinking: block.Thinking})
		}
	}
	return lines
}
func (v *View) Start(lines []Line, height int) int {
	if v.Follow {
		return max(0, len(lines)-height)
	}
	for i, line := range lines {
		if line.ID == v.AnchorID && line.Row >= v.AnchorRow {
			return min(i, max(0, len(lines)-height))
		}
	}
	return 0
}
func (v *View) Scroll(n, width, height int) {
	lines := v.Lines(width)
	if len(lines) == 0 {
		return
	}
	index := max(0, min(max(0, len(lines)-height), v.Start(lines, height)+n))
	v.Follow = index >= max(0, len(lines)-height)
	v.AnchorID, v.AnchorRow = lines[index].ID, lines[index].Row
}

func (v *View) Prepend(blocks []Block) {
	if len(blocks) == 0 {
		return
	}
	first := 0
	if len(v.Blocks) > 0 {
		first = v.Blocks[0].ID
	}
	for i := range blocks {
		blocks[i].ID = first - len(blocks) + i
	}
	v.Blocks = append(blocks, v.Blocks...)
	v.bound(true)
}
func (v *View) bound(older bool) {
	bytes := 0
	for _, b := range v.Blocks {
		bytes += len(b.Text)
	}
	for len(v.Blocks) > 1 && (len(v.Blocks) > 1000 || bytes > 8*1024*1024) {
		index := 0
		if older {
			index = len(v.Blocks) - 1
		} else if !v.Follow {
			index = len(v.Blocks) - 2
		}
		bytes -= len(v.Blocks[index].Text)
		copy(v.Blocks[index:], v.Blocks[index+1:])
		v.Blocks = v.Blocks[:len(v.Blocks)-1]
	}
}
