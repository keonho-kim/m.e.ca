// Purpose: Exclude shared whole-line prefixes and suffixes before allocating diff state.
// Pattern: Pure range calculation.
// Usage: Exact counts omit unchanged lines; previews retain their nearest context lines.
// Related: internal/tools/local/diff/diff.go, internal/tools/local/diff/count.go.
package diff

import "bytes"

func unchangedRegion(before, after []byte) (prefix, suffix int) {
	// Equal blocks use bytes.Equal's native vector comparison; scan individual
	// bytes only near the first and last differences.
	const block = 4096
	limit := min(len(before), len(after))
	for limit-prefix >= block && bytes.Equal(before[prefix:prefix+block], after[prefix:prefix+block]) {
		prefix += block
	}
	for prefix < min(len(before), len(after)) && before[prefix] == after[prefix] {
		prefix++
	}
	prefix = bytes.LastIndexByte(before[:prefix], '\n') + 1
	before, after = before[prefix:], after[prefix:]
	limit = min(len(before), len(after))
	for limit-suffix >= block && bytes.Equal(before[len(before)-suffix-block:len(before)-suffix], after[len(after)-suffix-block:len(after)-suffix]) {
		suffix += block
	}
	for suffix < min(len(before), len(after)) && before[len(before)-1-suffix] == after[len(after)-1-suffix] {
		suffix++
	}
	if suffix > 0 {
		start := len(before) - suffix
		// A common byte suffix can begin inside a changed line.
		if start > 0 && before[start-1] != '\n' || len(after) > suffix && after[len(after)-suffix-1] != '\n' {
			if newline := bytes.IndexByte(before[start:], '\n'); newline >= 0 {
				suffix -= newline + 1
			} else {
				suffix = 0
			}
		}
	}
	return prefix, suffix
}

func contextRegion(before, after []byte) ([]byte, []byte, int) {
	start, suffix := unchangedRegion(before, after)
	for range contextLines {
		if start == 0 {
			break
		}
		start = bytes.LastIndexByte(before[:start-1], '\n') + 1
	}
	end := len(before) - suffix
	remaining := before[end:]
	for range contextLines {
		if len(remaining) == 0 {
			break
		}
		n := bytes.IndexByte(remaining, '\n') + 1
		if n == 0 {
			n = len(remaining)
		}
		suffix -= n
		remaining = remaining[n:]
	}
	return before[start : len(before)-suffix], after[start : len(after)-suffix], lineCount(before[:start])
}
