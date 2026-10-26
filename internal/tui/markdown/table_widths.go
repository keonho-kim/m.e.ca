// Purpose: Allocate table column widths from content size and the available terminal cells.
// Pattern: Pure layout calculation with bounded water filling.
// Usage: table requests widths before wrapping and composing individual cells.
// Related: internal/tui/markdown/table.go, internal/tui/markdown/layout.go.
package markdown

import (
	"strings"

	"github.com/rivo/uniseg"
)

func tableWidths(rows [][]tableCell, width int) []int {
	var natural []int
	for _, row := range rows {
		for i, cell := range row {
			if i >= len(natural) {
				natural = append(natural, 1)
			}
			for part := range strings.SplitSeq(line(cell.spans).Text, "\n") {
				natural[i] = max(natural[i], uniseg.StringWidth(part))
			}
		}
	}
	if len(natural) == 0 {
		return nil
	}
	available := width - 3*(len(natural)-1)
	minimum := 0
	for _, size := range natural {
		minimum += min(4, size)
	}
	if available < minimum {
		return nil
	}
	// Cap long columns equally; short columns leave their unused share for prose.
	low, high := 1, available
	for low < high {
		cap := low + (high-low+1)/2
		used := 0
		for _, size := range natural {
			used += min(size, cap)
		}
		if used <= available {
			low = cap
		} else {
			high = cap - 1
		}
	}
	used := 0
	widths := make([]int, len(natural))
	for i, size := range natural {
		widths[i] = min(size, low)
		used += widths[i]
	}
	// At most one extra cell per capped column remains after the binary search.
	for i, size := range natural {
		if used < available && widths[i] < size {
			widths[i]++
			used++
		}
	}
	return widths
}
