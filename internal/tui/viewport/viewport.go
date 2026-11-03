// Purpose: Keep bounded output blocks with source-bound formatting and event-relative reading anchors.
// Pattern: Simple Module (presentation projection).
// Usage: Mutated only by the terminal event loop and rendered at the current width.
// Related: internal/tui/viewport/changes.go, internal/tui/viewport/block_layout.go, internal/tui/session/render_chat.go.
package viewport

import (
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/tui/markdown"
	"github.com/keonho-kim/meca/internal/tui/textlayout"
	"strings"
	"unsafe"
)

const sourceMemoryBudget = 16 * 1024 * 1024

type Block struct {
	Summary, Details string
	DetailRef        *runrecord.EventReference
	Changes          *Changes
	stream           markdown.Stream
	layout           blockLayout
	visible          bool
	live             bool
	Markdown         bool
	Streaming        bool
	Centered         bool
	ID               int
	Text             string
	Thinking         bool
	KeepWithNext     bool // Retain an attribution header together with its following body.
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
	dirtyID             int
	sourceBytes         int
	bytesReady          bool
	streams             map[int]bool
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
	v.ensureBytes()
	v.showDetails(&block)
	if strings.TrimSpace(block.Text) == "" {
		return
	}
	v.Next++
	block.ID = v.Next
	v.Blocks = append(v.Blocks, block)
	v.sourceBytes += block.bytes()
	v.trackStream(block.ID, block.Streaming)
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
	v.ensureBytes()
	first := 0
	if len(v.Blocks) > 0 {
		first = v.Blocks[0].ID
	}
	for i := range blocks {
		blocks[i].ID = first - len(blocks) + i
		v.showDetails(&blocks[i])
		v.sourceBytes += blocks[i].bytes()
		v.trackStream(blocks[i].ID, blocks[i].Streaming)
	}
	v.Blocks = append(blocks, v.Blocks...)
	v.bound(true)
}
func (v *View) bound(older bool) {
	v.ensureBytes()
	if v.sourceBytes > sourceMemoryBudget {
		v.releaseColdDetails()
	}
	for len(v.Blocks) > 1 && (len(v.Blocks) > 1000 || v.sourceBytes > sourceMemoryBudget) {
		index := 0
		if older {
			index = len(v.Blocks) - 1
		} else if !v.Follow {
			index = len(v.Blocks) - 2
		}
		first, last := v.group(index)
		// Keep the newest complete message while reading older history, just
		// as an ungrouped last block remains available under memory pressure.
		if !older && last == len(v.Blocks) && first > 0 {
			first, last = v.group(first - 1)
		}
		if first == 0 && last == len(v.Blocks) {
			break
		}
		v.layoutRevision++
		for _, block := range v.Blocks[first:last] {
			v.sourceBytes -= block.bytes()
			delete(v.streams, block.ID)
		}
		copy(v.Blocks[first:], v.Blocks[last:])
		size := len(v.Blocks) - (last - first)
		clear(v.Blocks[size:])
		v.Blocks = v.Blocks[:size]
	}
}

func (v *View) group(index int) (first, last int) {
	first, last = index, index+1
	for first > 0 && v.Blocks[first-1].KeepWithNext {
		first--
	}
	for last < len(v.Blocks) && v.Blocks[last-1].KeepWithNext {
		last++
	}
	return first, last
}

func (v *View) ensureBytes() {
	if !v.bytesReady {
		for _, b := range v.Blocks {
			v.sourceBytes += b.bytes()
		}
		v.bytesReady = true
	}
}

func (b Block) bytes() int {
	size := len(b.Text)
	if b.Details != "" {
		size = len(b.Summary) + len(b.Details)
	}
	if b.Changes != nil {
		size += cap(b.Changes.Ranges) * int(unsafe.Sizeof(ChangeRange{}))
		if b.Changes.Source != b.Text && b.Changes.Source != b.Details && b.Changes.Source != b.Summary {
			size += len(b.Changes.Source)
		}
	}
	return size
}
