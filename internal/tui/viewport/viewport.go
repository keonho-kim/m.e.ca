// Purpose: Keep bounded output blocks and preserve an event-relative reading anchor on resize.
// Pattern: Simple Module (presentation projection).
// Usage: Mutated only by the terminal event loop and rendered at the current width.
// Related: internal/tui/viewport/spacing.go, internal/tui/session/render_chat.go.
package viewport

import (
	"github.com/keonho-kim/v-code/internal/tui/markdown"
	"github.com/keonho-kim/v-code/internal/tui/textlayout"
	"strings"
)

type Block struct {
	Summary, Details string
	key              blockKey
	lines            []Line
	parsedText       string
	parsed           []markdown.Line
	Markdown         bool
	ID               int
	Text             string
	Thinking         bool
}
type Line struct {
	Copy     textlayout.Fragment
	Spans    []markdown.Span
	ID, Row  int
	Text     string
	Thinking bool
}
type View struct {
	Expanded            bool
	keys                []blockKey
	lines               []Line
	Blocks              []Block
	Next                int
	Follow              bool
	AnchorID, AnchorRow int
}

type blockKey struct {
	text               string
	width, id          int
	markdown, thinking bool
}

func (v *View) Append(text string, thinking bool) {
	v.AppendBlock(Block{Text: text, Thinking: thinking})
}
func (v *View) AppendBlock(block Block) {
	if strings.TrimSpace(block.Text) == "" {
		return
	}
	if block.Details != "" {
		block.Summary = block.Text
		if v.Expanded {
			block.Text = block.Details
		}
	}
	v.Next++
	block.ID = v.Next
	v.Blocks = append(v.Blocks, block)
	v.bound(false)
}
func (b *Block) Lines(width int) []Line {
	key := blockKey{b.Text, width, b.ID, b.Markdown, b.Thinking}
	if b.lines != nil && b.key == key {
		return b.lines
	}
	b.key = key
	if strings.TrimSpace(b.Text) == "" {
		b.lines = nil
		b.parsed, b.parsedText = nil, ""
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
				value := Line{ID: b.ID, Row: len(out), Text: wrapped.Text, Spans: wrapped.Spans, Thinking: b.Thinking, Copy: wrapped.Copy}
				out = append(out, value)
			}
		}
	} else {
		for row, text := range textlayout.Wrap(b.Text, width) {
			out = append(out, Line{ID: b.ID, Row: row, Text: text.Text, Thinking: b.Thinking, Copy: textlayout.Fragment{Source: b.Text, Row: text}})
		}
	}
	if b.Markdown || b.Thinking {
		out = modelEdges(out)
	}
	b.lines = out
	return b.lines
}
func Wrap(text string, width int) []string {
	var out []string
	for _, row := range textlayout.Wrap(text, width) {
		out = append(out, row.Text)
	}
	return out
}
func (v *View) Lines(width int) []Line {
	changed := len(v.keys) != len(v.Blocks)
	if !changed {
		for i, b := range v.Blocks {
			if v.keys[i] != (blockKey{b.Text, width, b.ID, b.Markdown, b.Thinking}) {
				changed = true
				break
			}
		}
	}
	if !changed {
		return v.lines
	}
	var lines []Line
	v.keys = make([]blockKey, len(v.Blocks))
	for i := range v.Blocks {
		lines = AppendLines(lines, v.Blocks[i].Lines(width))
		v.keys[i] = v.Blocks[i].key
	}
	v.lines = lines
	return v.lines
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
		if blocks[i].Details != "" {
			blocks[i].Summary = blocks[i].Text
			if v.Expanded {
				blocks[i].Text = blocks[i].Details
			}
		}
	}
	v.Blocks = append(blocks, v.Blocks...)
	v.bound(true)
}
func (v *View) bound(older bool) {
	bytes := 0
	for _, b := range v.Blocks {
		bytes += b.bytes()
	}
	for len(v.Blocks) > 1 && (len(v.Blocks) > 1000 || bytes > 16*1024*1024) {
		index := 0
		if older {
			index = len(v.Blocks) - 1
		} else if !v.Follow {
			index = len(v.Blocks) - 2
		}
		bytes -= v.Blocks[index].bytes()
		copy(v.Blocks[index:], v.Blocks[index+1:])
		v.Blocks[len(v.Blocks)-1] = Block{}
		v.Blocks = v.Blocks[:len(v.Blocks)-1]
	}
}

func (v *View) HasDetails() bool {
	for _, block := range v.Blocks {
		if block.Details != "" {
			return true
		}
	}
	return false
}

func (v *View) ToggleDetails() bool {
	if !v.HasDetails() {
		return false
	}
	v.Expanded = !v.Expanded
	for i := range v.Blocks {
		b := &v.Blocks[i]
		if b.Details != "" {
			b.Text = b.Summary
			if v.Expanded {
				b.Text = b.Details
			}
			if b.ID == v.AnchorID {
				v.AnchorRow = 0
			}
		}
	}
	return true
}

func (b Block) bytes() int {
	if b.Details != "" {
		return len(b.Summary) + len(b.Details)
	}
	return len(b.Text)
}
