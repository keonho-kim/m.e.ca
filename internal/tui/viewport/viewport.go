// Purpose: Keep bounded output blocks and preserve an event-relative reading anchor on resize.
// Pattern: Simple Module (presentation projection).
// Usage: Mutated only by the terminal event loop and rendered at the current width.
// Related: internal/tui/viewport/spacing.go, internal/tui/session/render_chat.go.
package viewport

import (
	"github.com/keonho-kim/v-code/internal/tui/markdown"
	"github.com/rivo/uniseg"
	"strings"
)

type Block struct {
	parsedText string
	parsed     []markdown.Line
	Markdown   bool
	ID         int
	Text       string
	Thinking   bool
}
type Line struct {
	Spans    []markdown.Span
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
	v.AppendBlock(Block{Text: text, Thinking: thinking})
}
func (v *View) AppendBlock(block Block) {
	if strings.TrimSpace(block.Text) == "" {
		return
	}
	v.Next++
	block.ID = v.Next
	v.Blocks = append(v.Blocks, block)
	v.bound(false)
}
func (b *Block) Lines(width int) []Line {
	if strings.TrimSpace(b.Text) == "" {
		return nil
	}
	var out []Line
	if b.Markdown {
		if b.parsed == nil || b.parsedText != b.Text {
			b.parsed = markdown.Render(b.Text)
			b.parsedText = b.Text
		}
		for _, line := range b.parsed {
			for _, wrapped := range markdown.Wrap(line.Spans, width) {
				out = append(out, Line{ID: b.ID, Row: len(out), Text: wrapped.Text, Spans: wrapped.Spans, Thinking: b.Thinking})
			}
		}
	} else {
		for row, text := range Wrap(b.Text, width) {
			out = append(out, Line{ID: b.ID, Row: row, Text: text, Thinking: b.Thinking})
		}
	}
	if b.Markdown || b.Thinking {
		return modelEdges(out)
	}
	return out
}
func Wrap(text string, width int) []string {
	if width < 1 {
		return nil
	}
	var out []string
	for _, paragraph := range strings.Split(strings.ReplaceAll(text, "\t", "    "), "\n") {
		indent := paragraph[:len(paragraph)-len(strings.TrimLeft(paragraph, " "))]
		// Leave room for a two-cell character on indented continuation rows.
		indent = indent[:min(len(indent), max(0, width-2))]
		var line strings.Builder
		line.Grow(min(len(paragraph), width))
		cells := 0
		g := uniseg.NewGraphemes(paragraph)
		for g.Next() {
			part := g.Str()
			n := uniseg.StringWidth(part)
			if cells+n > width && line.Len() > 0 {
				out = append(out, line.String())
				line.Reset()
				line.Grow(min(len(paragraph), width))
				line.WriteString(indent)
				cells = len(indent)
			}
			line.WriteString(part)
			cells += n
		}
		out = append(out, line.String())
	}
	return out
}
func (v *View) Lines(width int) []Line {
	var lines []Line
	for i := range v.Blocks {
		lines = AppendLines(lines, v.Blocks[i].Lines(width))
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
		v.Blocks[len(v.Blocks)-1] = Block{}
		v.Blocks = v.Blocks[:len(v.Blocks)-1]
	}
}
