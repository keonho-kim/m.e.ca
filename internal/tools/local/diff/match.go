// Purpose: Match line changes with bounded trace memory and a complete preview fallback.
// Pattern: Bounded Myers line matching.
// Usage: Preview construction uses the exact flag to reuse counts or request an exact recount.
// Related: internal/tools/local/diff/diff.go, internal/tools/local/diff/count.go.
package diff

func match(before, after []string) ([]line, bool) {
	n, m := len(before), len(after)
	if n == 0 || m == 0 {
		return expanded(before, after), true
	}
	var trace [][]int
	work := 0
	for d := 0; d <= min(n+m, 256); d++ {
		current := make([]int, 2*d+1)
		for k := -d; k <= d; k += 2 {
			x := 0
			if d > 0 {
				prev := trace[d-1]
				if k == -d || k != d && prev[k-1+d-1] < prev[k+1+d-1] {
					x = prev[k+1+d-1]
				} else {
					x = prev[k-1+d-1] + 1
				}
			}
			y := x - k
			work++
			for x < n && y < m && before[x] == after[y] {
				x++
				y++
				work++
				if work > 1000000 {
					return expanded(before, after), false
				}
			}
			if work > 1000000 {
				return expanded(before, after), false
			}
			current[k+d] = x
			if x >= n && y >= m {
				trace = append(trace, current)
				return backtrack(before, after, trace), true
			}
		}
		trace = append(trace, current)
	}
	return expanded(before, after), false
}
func backtrack(before, after []string, trace [][]int) []line {
	x, y := len(before), len(after)
	var reversed []line
	for d := len(trace) - 1; d > 0; d-- {
		k, prev := x-y, trace[d-1]
		previousK := k - 1
		if k == -d || k != d && prev[k-1+d-1] < prev[k+1+d-1] {
			previousK = k + 1
		}
		previousX := prev[previousK+d-1]
		previousY := previousX - previousK
		for x > previousX && y > previousY {
			x--
			y--
			reversed = append(reversed, line{' ', before[x]})
		}
		if x == previousX {
			y--
			reversed = append(reversed, line{'+', after[y]})
		} else {
			x--
			reversed = append(reversed, line{'-', before[x]})
		}
	}
	for x > 0 && y > 0 {
		x--
		y--
		reversed = append(reversed, line{' ', before[x]})
	}
	for i, j := 0, len(reversed)-1; i < j; i, j = i+1, j-1 {
		reversed[i], reversed[j] = reversed[j], reversed[i]
	}
	return reversed
}
func expanded(before, after []string) []line {
	// A bounded fallback remains a complete valid diff; never silently omit changes.
	ops := make([]line, 0, len(before)+len(after))
	for _, text := range before {
		ops = append(ops, line{'-', text})
	}
	for _, text := range after {
		ops = append(ops, line{'+', text})
	}
	return ops
}
