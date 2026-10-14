// Purpose: Count the shortest line edit delta without substituting whole-file preview fallbacks.
// Pattern: Bounded Myers distance calculation.
// Usage: Tool preparation computes Conclave size from frozen before and after bytes.
// Related: internal/tools/local/diff/diff.go, internal/tools/local/review.go.
package diff

import (
	"bytes"
	"fmt"
)

func Count(before, after []byte) (added, deleted int, err error) {
	if bytes.Equal(before, after) {
		return 0, 0, nil
	}
	// Remove common complete lines before allocating the line slices.
	prefix := 0
	for prefix < min(len(before), len(after)) && before[prefix] == after[prefix] {
		prefix++
	}
	prefix = bytes.LastIndexByte(before[:prefix], '\n') + 1
	before, after = before[prefix:], after[prefix:]
	suffix := 0
	for suffix < min(len(before), len(after)) && before[len(before)-1-suffix] == after[len(after)-1-suffix] {
		suffix++
	}
	if suffix > 0 {
		start := len(before) - suffix
		// A shared byte suffix may start in the middle of a changed line.
		if start > 0 && before[start-1] != '\n' || len(after) > suffix && after[len(after)-suffix-1] != '\n' {
			if newline := bytes.IndexByte(before[start:], '\n'); newline >= 0 {
				suffix -= newline + 1
			} else {
				suffix = 0
			}
		}
		before, after = before[:len(before)-suffix], after[:len(after)-suffix]
	}
	n, m := lineCount(before), lineCount(after)
	if n == 0 || m == 0 {
		return m, n, nil
	}
	if n+m > 524288 {
		return 0, 0, fmt.Errorf("exact change count exceeds line budget; prepare a smaller change")
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
					return 0, 0, fmt.Errorf("exact change count exceeds work budget; prepare a smaller change")
				}
			}
			if work > 16000000 {
				return 0, 0, fmt.Errorf("exact change count exceeds work budget; prepare a smaller change")
			}
			frontier[offset+diagonal] = x
			if x >= n && y >= m {
				// added-deleted=m-n and added+deleted=distance.
				added = (distance + m - n) / 2
				return added, distance - added, nil
			}
		}
	}
	return 0, 0, fmt.Errorf("could not count line changes")
}
