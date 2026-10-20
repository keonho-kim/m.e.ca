// Purpose: Track reading positions against the complete displayed transcript, including live output.
// Pattern: Presentation state transitions.
// Usage: Session wheel, page keys and scrollbar pass the same projected lines used for rendering.
// Related: internal/tui/viewport/viewport.go, internal/tui/session/history.go.
package viewport

func (v *View) Start(lines []Line, height int) int {
	last := max(0, len(lines)-max(1, height))
	if v.Follow {
		return last
	}
	for i, line := range lines {
		if line.ID > v.AnchorID || line.ID == v.AnchorID && line.Row >= v.AnchorRow {
			return min(i, last)
		}
	}
	return last
}

func (v *View) Scroll(lines []Line, n, height int) {
	v.Seek(lines, v.Start(lines, height)+n, height)
}

func (v *View) Seek(lines []Line, index, height int) {
	last := max(0, len(lines)-max(1, height))
	index = max(0, min(last, index))
	v.Follow = index == last
	if len(lines) > 0 {
		v.AnchorID, v.AnchorRow = lines[index].ID, lines[index].Row
	}
}
