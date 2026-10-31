// Purpose: Provide full-layout reference projections for differential sparse-window tests.
// Pattern: Test reference implementation.
// Usage: Viewport tests compare indexed windows to independently composed complete layouts.
// Related: internal/tui/viewport/block_layout.go, internal/tui/markdown/render.go.
package viewport

import (
	"github.com/keonho-kim/meca/internal/tui/markdown"
	"github.com/keonho-kim/meca/internal/tui/textlayout"
	"strings"
)

func (b *Block) Lines(width int) []Line {
	if strings.TrimSpace(b.Text) == "" {
		return nil
	}
	var out []Line
	if b.Markdown {
		for _, line := range markdown.Render(b.Text, width) {
			for wrapped := range markdown.Rows(line.Spans, width) {
				value := Line{ID: b.ID, Row: len(out), Text: wrapped.Text, Spans: wrapped.Spans, Thinking: b.Thinking, Copy: wrapped.Copy}
				out = append(out, value)
			}
		}
	} else {
		out = make([]Line, 0, strings.Count(b.Text, "\n")+1)
		for text := range textlayout.Rows(b.Text, width) {
			row := len(out)
			out = append(out, Line{ID: b.ID, Row: row, Text: text.Text, Thinking: b.Thinking, Copy: textlayout.Fragment{Source: b.Text, Row: text}})
		}
		b.classifyChanges(out)
	}
	if b.Markdown || b.Thinking {
		out = modelEdges(out)
	}
	if b.Centered {
		centerLines(out, width)
	}
	return out
}
func blank(line Line) bool { return strings.TrimSpace(line.Text) == "" }

// AppendLines removes only excess leading separator rows from the incoming block.
// Interior rows and existing literal tool/file output remain untouched.
func AppendLines(previous, next []Line) []Line {
	trailing := 0
	for i := len(previous) - 1; i >= 0 && blank(previous[i]); i-- {
		trailing++
		if trailing >= maxBoundaryBlanks {
			break
		}
	}
	leading := 0
	for leading < len(next) && blank(next[leading]) {
		leading++
	}
	skip := max(0, leading-max(0, maxBoundaryBlanks-trailing))
	return append(previous, next[skip:]...)
}

func modelEdges(lines []Line) []Line {
	for len(lines) > 0 && blank(lines[0]) {
		lines = lines[1:]
	}
	end := len(lines)
	for end > 0 && blank(lines[end-1]) {
		end--
	}
	return lines[:min(len(lines), end+maxBoundaryBlanks)]
}

func (t *Transcript) all(v *View, width int) []Line {
	t.Prepare(width, v, nil)
	return t.Window(0, t.Count())
}
