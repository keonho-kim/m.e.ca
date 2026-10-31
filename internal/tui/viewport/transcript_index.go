// Purpose: Extend the immutable history index while rebuilding only the small live transcript suffix.
// Pattern: Event-loop-owned incremental prefix projection.
// Usage: Transcript.Prepare reads View mutations and resets on reflow, prepend, trimming or detail changes.
// Related: internal/tui/viewport/window.go, internal/tui/viewport/viewport.go, internal/tui/viewport/details.go.
package viewport

// Prepare consumes history owned by View; callers change its blocks through
// AppendBlock, Prepend and ToggleDetails so unchanged prefixes remain reusable.
func (t *Transcript) Prepare(width int, view *View, live []*Block) {
	history := view.Blocks
	var head *Block
	if len(history) > 0 {
		head = &history[0]
	}
	reset := t.width != width || t.historyHead != head || t.historyRevision != view.layoutRevision || len(history) < t.historyLength
	changed := reset || len(history) != t.historyLength
	if reset {
		for _, block := range t.visible {
			block.visible = false
			block.releaseParsed()
		}
		clear(t.visible)
		t.visible = t.visible[:0]
		for i := range history {
			history[i].visible = false
		}
		clear(t.sections)
		t.sections = t.sections[:0]
		t.historyLength, t.historySections, t.historyRows, t.historyTrailing = 0, 0, 0, 0
	}
	previous := t.sections
	t.sections = t.sections[:t.historySections]
	t.width, t.total = width, t.historyRows
	trailing := t.historyTrailing
	add := func(block *Block) {
		count := block.Count(width)
		if count == 0 {
			return
		}
		skip := max(0, block.layout.leading-max(0, maxBoundaryBlanks-trailing))
		part := section{block: block, key: block.layout.key, start: t.total, skip: skip, count: count - skip}
		i := len(t.sections)
		if i >= len(previous) || previous[i] != part {
			changed = true
		}
		t.sections = append(t.sections, part)
		t.total += part.count
		trailing = min(maxBoundaryBlanks, block.layout.trailing)
	}
	for i := t.historyLength; i < len(history); i++ {
		history[i].live = false
		add(&history[i])
		if !history[i].visible {
			history[i].releaseParsed()
		}
	}
	t.historyHead, t.historyRevision, t.historyLength = head, view.layoutRevision, len(history)
	t.historySections, t.historyRows, t.historyTrailing = len(t.sections), t.total, trailing
	for _, block := range live {
		block.live = true
		add(block)
	}
	if len(previous) != len(t.sections) {
		changed = true
	}
	if len(previous) > len(t.sections) {
		clear(previous[len(t.sections):])
	}
	if changed {
		t.rows = nil
	}
}
