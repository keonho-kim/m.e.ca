// Purpose: Keep tool details expanded by default and toggle only their presentation.
// Pattern: Presentation state transition.
// Usage: AppendBlock, Prepend and terminal Ctrl+O share this session-owned state.
// Related: internal/tui/viewport/viewport.go, internal/tui/transcript/events.go.
package viewport

func (v *View) showDetails(block *Block) {
	if block.Details != "" {
		block.Text = block.Details
		if v.Collapsed {
			block.Text = block.Summary
		}
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
	v.Collapsed = !v.Collapsed
	v.layoutRevision++
	for i := range v.Blocks {
		block := &v.Blocks[i]
		v.showDetails(block)
		if block.Details != "" && block.ID == v.AnchorID {
			v.AnchorRow = 0
		}
	}
	return true
}
