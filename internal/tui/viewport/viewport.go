// Purpose: Keep bounded output blocks with source-bound formatting and event-relative reading anchors.
// Pattern: Simple Module (presentation projection).
// Usage: Mutated only by the terminal event loop and rendered at the current width.
// Related: internal/tui/viewport/changes.go, internal/tui/viewport/block_layout.go, internal/tui/session/render_chat.go.
package viewport

import (
	"github.com/keonho-kim/meca/internal/tui/markdown"
	"github.com/keonho-kim/meca/internal/tui/textlayout"
	"strings"
)

type Block struct {
	Summary, Details string
	Changes          *Changes
	stream           markdown.Stream
	layout           blockLayout
	visible          bool
	Markdown         bool
	Centered         bool
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
	Change   byte
}
type View struct {
	layoutRevision      uint64
	Collapsed           bool
	Blocks              []Block
	Next                int
	Follow              bool
	AnchorID, AnchorRow int
}

type blockKey struct {
	text                         string
	width, id                    int
	markdown, thinking, centered bool
	changes                      *Changes
}

func (v *View) Append(text string, thinking bool) {
	v.AppendBlock(Block{Text: text, Thinking: thinking})
}
func (v *View) AppendBlock(block Block) {
	v.showDetails(&block)
	if strings.TrimSpace(block.Text) == "" {
		return
	}
	v.Next++
	block.ID = v.Next
	v.Blocks = append(v.Blocks, block)
	v.bound(false)
}
func Wrap(text string, width int) []string {
	var out []string
	for _, row := range textlayout.Wrap(text, width) {
		out = append(out, row.Text)
	}
	return out
}
func (v *View) Prepend(blocks []Block) {
	if len(blocks) == 0 {
		return
	}
	v.layoutRevision++
	first := 0
	if len(v.Blocks) > 0 {
		first = v.Blocks[0].ID
	}
	for i := range blocks {
		blocks[i].ID = first - len(blocks) + i
		v.showDetails(&blocks[i])
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
		v.layoutRevision++
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

func (b Block) bytes() int {
	if b.Details != "" {
		return len(b.Summary) + len(b.Details)
	}
	return len(b.Text)
}
