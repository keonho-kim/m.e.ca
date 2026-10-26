// Purpose: Seek Unicode grapheme boundaries from sparse reusable parser checkpoints.
// Pattern: Editor-owned incremental index.
// Usage: Navigation reads checkpoints; text replacement discards the affected suffix.
// Related: internal/tui/input/editor.go, internal/tui/input/navigation.go.
package input

import (
	"slices"
	"sort"
	"unicode/utf8"

	"github.com/rivo/uniseg"
)

const boundaryStride = 256

type boundaryPoint struct{ rune, byte, state int }

func (e *Editor) invalidateBoundaries(start int) {
	// Parser state includes the first rune after the preceding cluster. Retain
	// only checkpoints strictly before the edit, including insertions at EOF.
	end := sort.Search(len(e.boundaries), func(i int) bool { return e.boundaries[i].rune >= start })
	e.boundaries = e.boundaries[:end]
	if end == 0 {
		e.boundaries = nil
	} else if cap(e.boundaries) > 4*end {
		// A large deletion must release the index storage for its removed tail.
		e.boundaries = slices.Clone(e.boundaries)
	}
}

func (e *Editor) boundary(position int, next bool) int {
	point := boundaryPoint{state: -1}
	index := sort.Search(len(e.boundaries), func(i int) bool { return e.boundaries[i].rune >= position })
	if index > 0 {
		point = e.boundaries[index-1]
	}
	text := e.String()
	for point.byte < len(text) {
		cluster, _, _, state := uniseg.FirstGraphemeClusterInString(text[point.byte:], point.state)
		end := boundaryPoint{rune: point.rune + utf8.RuneCountInString(cluster), byte: point.byte + len(cluster), state: state}
		last := 0
		if len(e.boundaries) > 0 {
			last = e.boundaries[len(e.boundaries)-1].rune
		}
		if end.rune-last >= boundaryStride {
			e.boundaries = append(e.boundaries, end)
		}
		if next && end.rune > position {
			return end.rune
		}
		if !next && end.rune >= position {
			return point.rune
		}
		point = end
	}
	return len(e.runes)
}
