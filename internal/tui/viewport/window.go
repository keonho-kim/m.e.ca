// Purpose: Compose sparse block layouts into a scrollable transcript without retaining all wrapped rows.
// Pattern: Indexed presentation projection.
// Usage: Session owns Transcript and requests only the visible window for rendering and copying.
// Related: internal/tui/viewport/block_layout.go, internal/tui/session/chat_projection.go.
package viewport

import "sort"

type section struct {
	block              *Block
	key                blockKey
	start, skip, count int
}
type Transcript struct {
	sections                 []section
	width, total             int
	rows                     []Line
	windowStart, windowCount int
}

func (t *Transcript) Prepare(width int, history []Block, live []*Block) {
	previous := t.sections
	changed := t.width != width
	t.sections = t.sections[:0]
	t.width, t.total = width, 0
	trailing := 0
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
		t.total += count - skip
		trailing = min(maxBoundaryBlanks, block.layout.trailing)
	}
	for i := range history {
		add(&history[i])
		if !history[i].visible {
			history[i].releaseParsed()
		}
	}
	for _, block := range live {
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

func (t *Transcript) Count() int { return t.total }
func (t *Transcript) Window(start, count int) []Line {
	start = max(0, start)
	count = min(count, max(0, t.total-start))
	if t.rows != nil && t.windowStart == start && t.windowCount == count {
		return t.rows
	}
	for _, part := range t.sections {
		visible := count > 0 && part.start < start+count && part.start+part.count > start
		if part.block.visible && !visible {
			part.block.releaseParsed()
		}
		part.block.visible = visible
	}
	if count < 1 || start >= t.total {
		t.rows = nil
		return nil
	}
	count = min(count, t.total-start)
	at := sort.Search(len(t.sections), func(i int) bool { return t.sections[i].start+t.sections[i].count > start })
	rows := make([]Line, 0, count)
	for ; at < len(t.sections) && len(rows) < count; at++ {
		part := t.sections[at]
		offset := max(0, start-part.start)
		rows = append(rows, part.block.Window(t.width, part.skip+offset, min(count-len(rows), part.count-offset))...)
	}
	t.rows, t.windowStart, t.windowCount = rows, start, count
	return rows
}
func (t *Transcript) Start(v *View, height int) int {
	last := max(0, t.total-max(1, height))
	if v.Follow {
		return last
	}
	for _, part := range t.sections {
		if part.block.ID > v.AnchorID {
			return min(part.start, last)
		}
		if part.block.ID == v.AnchorID {
			local := v.AnchorRow - part.block.layout.trim - part.skip
			if local < part.count {
				return min(part.start+max(0, local), last)
			}
		}
	}
	return last
}
func (t *Transcript) Seek(v *View, index, height int) {
	last := max(0, t.total-max(1, height))
	index = max(0, min(last, index))
	v.Follow = index == last
	if rows := t.Window(index, 1); len(rows) > 0 {
		v.AnchorID, v.AnchorRow = rows[0].ID, rows[0].Row
	}
}
func (t *Transcript) Scroll(v *View, n, height int) { t.Seek(v, t.Start(v, height)+n, height) }
