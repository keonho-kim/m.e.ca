// Purpose: Keep one public response or tool card paired with its sender across empty prefixes and eviction.
// Pattern: Source-owned presentation transition.
// Usage: State.Apply and Commands update owned display pairs through messageDisplay.show.
// Related: internal/tui/chat/team/events.go, internal/tui/chat/team/tools.go, internal/tui/viewport/update.go.
package team

import (
	"sort"
	"strings"

	"github.com/keonho-kim/meca/internal/tui/transcript"
	"github.com/keonho-kim/meca/internal/tui/viewport"
)

type messageDisplay struct{ header, body int }

func (p *messageDisplay) show(agent string, final bool, body viewport.Block, f transcript.Formatter, output *viewport.View) {
	if strings.TrimSpace(body.Text) == "" {
		return
	}
	if p.paired(output) {
		if final {
			output.Update(p.header, f.TeamHeader(agent, final))
		}
		if output.Update(p.body, body) && p.paired(output) {
			return
		}
	}
	// Trimming can remove either owned block. Clear surviving fragments before
	// appending the complete snapshot with its own header at the current tail.
	if p.header != 0 {
		output.Update(p.header, viewport.Block{})
	}
	if p.body != 0 {
		output.FinishStream(p.body)
		output.Update(p.body, viewport.Block{})
	}
	p.header = appendBlock(output, f.TeamHeader(agent, final))
	p.body = appendBlock(output, body)
}

func (p *messageDisplay) paired(output *viewport.View) bool {
	if p.header == 0 || p.body == 0 {
		return false
	}
	i := sort.Search(len(output.Blocks), func(i int) bool { return output.Blocks[i].ID >= p.header })
	return i+1 < len(output.Blocks) && output.Blocks[i].ID == p.header && output.Blocks[i+1].ID == p.body
}
