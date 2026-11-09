// Purpose: Count the shortest line edit delta without substituting whole-file preview fallbacks.
// Pattern: Bounded Myers distance calculation.
// Usage: Tool preparation counts actual added and deleted lines for frozen approval candidates.
// Related: internal/tools/local/diff/diff.go, internal/tools/local/approval_candidate.go.
package diff

import (
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func Count(before, after []byte) (added, deleted int, err error) {
	start, suffix := unchangedRegion(before, after)
	before, after = before[start:len(before)-suffix], after[start:len(after)-suffix]

	n, m := lineCount(before), lineCount(after)
	if n == 0 || m == 0 {
		return m, n, nil
	}
	if n+m > 524288 {
		return 0, 0, faults.Errorf(diagcatalog.DiffExactChangeCountExceedsLineBudgetPrepare)
	}
	left, right := split(string(before)), split(string(after))
	frontier := make([]int, 2*(n+m)+3)
	offset := n + m + 1
	work := 0
	for distance := 0; distance <= n+m; distance++ {
		for diagonal := -distance; diagonal <= distance; diagonal += 2 {
			x := 0
			if diagonal == -distance || diagonal != distance && frontier[offset+diagonal-1] < frontier[offset+diagonal+1] {
				x = frontier[offset+diagonal+1]
			} else {
				x = frontier[offset+diagonal-1] + 1
			}
			y := x - diagonal
			work++
			for x < n && y < m && left[x] == right[y] {
				x++
				y++
				work++
				if work > 16000000 {
					return 0, 0, faults.Errorf(diagcatalog.DiffExactChangeCountExceedsWorkBudgetPrepare)
				}
			}
			if work > 16000000 {
				return 0, 0, faults.Errorf(diagcatalog.DiffExactChangeCountExceedsWorkBudgetPrepare)
			}
			frontier[offset+diagonal] = x
			if x >= n && y >= m {
				// added-deleted=m-n and added+deleted=distance.
				added = (distance + m - n) / 2
				return added, distance - added, nil
			}
		}
	}
	return 0, 0, faults.Errorf(diagcatalog.DiffCouldNotCountLineChanges)
}
