// Purpose: Update one stable live history block while preserving its incremental Markdown state.
// Pattern: Event-loop-owned mutation with suffix invalidation.
// Usage: IRC streams and command cards replace their own block by ID.
// Related: internal/tui/viewport/transcript_index.go, internal/tui/session/team_events.go.
package viewport

import "sort"

func (v *View) Update(id int, value Block) bool {
	v.ensureBytes()
	i := sort.Search(len(v.Blocks), func(i int) bool { return v.Blocks[i].ID >= id })
	if i < len(v.Blocks) {
		block := &v.Blocks[i]
		if block.ID != id {
			return false
		}
		wasStreaming := block.Streaming
		v.sourceBytes -= block.bytes()
		block.Text, block.Summary, block.Details = value.Text, value.Summary, value.Details
		block.Changes, block.Markdown, block.Thinking, block.Centered = value.Changes, value.Markdown, value.Thinking, value.Centered
		block.Streaming = value.Streaming
		if wasStreaming && !value.Streaming {
			block.live = false
			block.releaseParsed()
		}
		block.DetailRef = value.DetailRef
		v.showDetails(block)
		v.sourceBytes += block.bytes()
		v.trackStream(id, value.Streaming)
		if v.dirtyID == 0 || id < v.dirtyID {
			v.dirtyID = id
		}
		v.bound(false)
		return true
	}
	return false
}

func (t *Transcript) updateSuffix(view *View) {
	if view.dirtyID == 0 {
		return
	}
	index := sort.Search(len(view.Blocks), func(i int) bool { return view.Blocks[i].ID >= view.dirtyID })
	if index >= t.historyLength {
		return
	}
	section := sort.Search(t.historySections, func(i int) bool { return t.sections[i].block.ID >= view.dirtyID })
	t.historyLength, t.historySections = index, section
	t.historyRows, t.historyTrailing = 0, 0
	if section > 0 {
		previous := t.sections[section-1]
		t.historyRows = previous.start + previous.count
		t.historyTrailing = min(maxBoundaryBlanks, previous.block.layout.trailing)
	}
	t.rows = nil
}
