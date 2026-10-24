// Purpose: Preserve explicitly classified source changes while reflowing literal tool details.
// Pattern: Pure presentation projection.
// Usage: Transcript formatters attach immutable changes; Block.Window classifies visible wrapped rows.
// Related: internal/tui/transcript/change_preview.go, internal/tui/viewport/viewport.go.
package viewport

import "sort"

// ChangeRange marks one added or removed line by byte offsets in the displayed source.
type ChangeRange struct {
	Start, End int
	Kind       byte
}

// Changes is immutable after construction. Source binds its ranges to expanded details.
type Changes struct {
	Source string
	Ranges []ChangeRange
}

func (b *Block) classifyChanges(lines []Line) {
	if b.Changes == nil || b.Text != b.Changes.Source {
		return
	}
	ranges := b.Changes.Ranges
	if len(lines) > 0 {
		start := lines[0].Copy.Row.Start
		ranges = ranges[sort.Search(len(ranges), func(i int) bool { return ranges[i].End > start }):]
	}
	for i := range lines {
		row := lines[i].Copy.Row
		for len(ranges) > 0 && ranges[0].End <= row.Start {
			ranges = ranges[1:]
		}
		if len(ranges) > 0 && row.Start >= ranges[0].Start && row.Start < ranges[0].End {
			lines[i].Change = ranges[0].Kind
		}
	}
}
