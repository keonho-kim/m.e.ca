// Purpose: Format complete file changes as bounded line-level unified approval previews.
// Pattern: Pure Function (bounded Myers line matching).
// Usage: Tool preparation supplies the exact original and frozen replacement bytes.
// Related: internal/tools/local/diff/diff_test.go, internal/tools/local/executor.go.
package diff

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"strings"
	"unicode/utf8"
)

const contextLines = 3

type line struct {
	kind byte
	text string
}

func Preview(path string, before, after []byte, existed, exists bool) string {
	oldPath, newPath := path, path
	if !existed {
		oldPath = "/dev/null"
	}
	if !exists {
		newPath = "/dev/null"
	}
	var out strings.Builder
	fmt.Fprintf(&out, "--- %s\n+++ %s\n", oldPath, newPath)
	if bytes.Equal(before, after) {
		out.WriteString("No content changes.\n")
		return out.String()
	}
	if !utf8.Valid(before) || !utf8.Valid(after) || bytes.IndexByte(before, 0) >= 0 || bytes.IndexByte(after, 0) >= 0 {
		fmt.Fprintf(&out, "Binary content changed: %d bytes (%x) -> %d bytes (%x)\n", len(before), sha256.Sum256(before), len(after), sha256.Sum256(after))
		return out.String()
	}
	// Avoid allocating a slice for every line in exceptionally large line counts.
	if bytes.Count(before, []byte{'\n'}) > 65536 || bytes.Count(after, []byte{'\n'}) > 65536 {
		fmt.Fprintf(&out, "@@ -%s +%s @@\n", span(1, lineCount(before)), span(1, lineCount(after)))
		writeWhole(&out, '-', string(before))
		writeWhole(&out, '+', string(after))
		return out.String()
	}
	ops := match(split(string(before)), split(string(after)))
	oldLine, newLine := 1, 1
	for cursor := 0; cursor < len(ops); {
		change := cursor
		for change < len(ops) && ops[change].kind == ' ' {
			change++
		}
		if change == len(ops) {
			break
		}
		start := max(cursor, change-contextLines)
		for _, op := range ops[cursor:start] {
			if op.kind != '+' {
				oldLine++
			}
			if op.kind != '-' {
				newLine++
			}
		}
		last := change
		end := change + 1
		for end < len(ops) {
			if ops[end].kind != ' ' {
				last = end
			} else if end-last > 2*contextLines {
				break
			}
			end++
		}
		end = min(len(ops), last+contextLines+1)
		oldCount, newCount := 0, 0
		for _, op := range ops[start:end] {
			if op.kind != '+' {
				oldCount++
			}
			if op.kind != '-' {
				newCount++
			}
		}
		fmt.Fprintf(&out, "@@ -%s +%s @@\n", span(oldLine, oldCount), span(newLine, newCount))
		for _, op := range ops[start:end] {
			writeLine(&out, op.kind, op.text)
		}
		oldLine += oldCount
		newLine += newCount
		cursor = end
	}
	return out.String()
}
func split(text string) []string {
	if text == "" {
		return nil
	}
	lines := strings.SplitAfter(text, "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}
func lineCount(text []byte) int {
	n := bytes.Count(text, []byte{'\n'})
	if len(text) > 0 && text[len(text)-1] != '\n' {
		n++
	}
	return n
}
func span(start, count int) string {
	if count == 0 {
		start--
	}
	return fmt.Sprintf("%d,%d", start, count)
}
func writeLine(out *strings.Builder, kind byte, text string) {
	out.WriteByte(kind)
	// Keep CRLF changes visible after terminal-control sanitization.
	out.WriteString(strings.ReplaceAll(text, "\r", "\\r"))
	if !strings.HasSuffix(text, "\n") {
		out.WriteString("\n\\ No newline at end of file\n")
	}
}
func writeWhole(out *strings.Builder, kind byte, text string) {
	for part := range strings.SplitAfterSeq(text, "\n") {
		if part != "" {
			writeLine(out, kind, part)
		}
	}
}

func match(before, after []string) []line {
	n, m := len(before), len(after)
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
					return expanded(before, after)
				}
			}
			if work > 1000000 {
				return expanded(before, after)
			}
			current[k+d] = x
			if x >= n && y >= m {
				trace = append(trace, current)
				return backtrack(before, after, trace)
			}
		}
		trace = append(trace, current)
	}
	return expanded(before, after)
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
