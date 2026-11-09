// Purpose: Keep tool results compact and release on-demand detail bodies when collapsed.
// Pattern: Presentation state transition.
// Usage: AppendBlock, Prepend and terminal Ctrl+O share this preference, including live tool rows.
// Related: internal/tui/viewport/viewport.go, internal/tui/transcript/events.go.
package viewport

import (
	"sort"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/tui/markdown"
)

func (v *View) showDetails(block *Block) {
	if block.DetailRef != nil && v.Collapsed {
		block.releaseDetails()
	}
	if block.Details != "" {
		block.Text = block.Details
		if v.Collapsed {
			block.Text = block.Summary
		}
	}
}

func (b *Block) releaseDetails() {
	b.Details, b.Text = "", b.Summary
	b.Changes = nil
	b.layout = blockLayout{}
	b.stream = markdown.Stream{}
}

// Keep reloadable summaries before dropping whole history blocks. Visible
// details and the newest result keep their normal expanded presentation.
func (v *View) releaseColdDetails() {
	for i := 0; i < len(v.Blocks)-1 && v.sourceBytes > sourceMemoryBudget; i++ {
		block := &v.Blocks[i]
		if block.visible || block.Streaming || block.DetailRef == nil || block.Details == "" {
			continue
		}
		before := block.bytes()
		block.releaseDetails()
		v.sourceBytes += block.bytes() - before
		v.layoutRevision++
	}
}

func (v *View) HasDetails() bool {
	for _, block := range v.Blocks {
		if block.Details != "" || block.DetailRef != nil {
			return true
		}
	}
	return false
}

func (v *View) ToggleDetails() {
	v.ensureBytes()
	v.Collapsed = !v.Collapsed
	v.layoutRevision++
	for i := range v.Blocks {
		block := &v.Blocks[i]
		before := block.bytes()
		v.showDetails(block)
		v.sourceBytes += block.bytes() - before
		if block.Details != "" && block.ID == v.AnchorID {
			v.AnchorRow = 0
		}
	}
}

// PendingDetail returns the first visible original that expanded presentation needs.
func (t *Transcript) PendingDetail() (int, *runrecord.EventReference) {
	for _, block := range t.visible {
		if block.DetailRef != nil && block.Details == "" {
			return block.ID, block.DetailRef
		}
	}
	return 0, nil
}

// UpdateDetails applies an on-demand result only while its original reference still owns this block.
func (v *View) UpdateDetails(id int, reference runrecord.EventReference, detail Block) bool {
	index := sort.Search(len(v.Blocks), func(i int) bool { return v.Blocks[i].ID >= id })
	if index >= len(v.Blocks) || v.Blocks[index].ID != id {
		return false
	}
	block := v.Blocks[index]
	if block.DetailRef == nil || *block.DetailRef != reference {
		return false
	}
	block.Details, block.Changes = detail.Text, detail.Changes
	return v.Update(id, block)
}
