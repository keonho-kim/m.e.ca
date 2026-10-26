// Purpose: Format complete file changes as bounded line-level unified approval previews.
// Pattern: Pure Function (bounded Myers line matching).
// Usage: Tool preparation supplies the exact original and frozen replacement bytes.
// Related: internal/tools/local/diff/match.go, internal/tools/local/diff/region.go, internal/tools/local/candidate.go.
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

// Analyze shares exact line matching between the preview and approval metadata.
func Analyze(path string, before, after []byte, existed, exists bool) (string, int, int, error) {
	preview, added, deleted, exact := preview(path, before, after, existed, exists)
	if exact {
		return preview, added, deleted, nil
	}
	added, deleted, err := Count(before, after)
	return preview, added, deleted, err
}

func preview(path string, before, after []byte, existed, exists bool) (string, int, int, bool) {
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
		return out.String(), 0, 0, true
	}
	if !utf8.Valid(before) || !utf8.Valid(after) || bytes.IndexByte(before, 0) >= 0 || bytes.IndexByte(after, 0) >= 0 {
		fmt.Fprintf(&out, "Binary content changed: %d bytes (%x) -> %d bytes (%x)\n", len(before), sha256.Sum256(before), len(after), sha256.Sum256(after))
		return out.String(), 0, 0, false
	}
	before, after, skipped := contextRegion(before, after)
	n, m := lineCount(before), lineCount(after)
	// Avoid allocating a slice for every line in exceptionally large line counts.
	// Complete creation/deletion also needs no matching trace or line slice.
	if n == 0 || m == 0 || n > 65536 || m > 65536 {
		fmt.Fprintf(&out, "@@ -%s +%s @@\n", span(skipped+1, n), span(skipped+1, m))
		writeWhole(&out, '-', string(before))
		writeWhole(&out, '+', string(after))
		return out.String(), m, n, n == 0 || m == 0
	}
	ops, exact := match(split(string(before)), split(string(after)))
	added, deleted := 0, 0
	for _, op := range ops {
		if op.kind == '+' {
			added++
		} else if op.kind == '-' {
			deleted++
		}
	}
	oldLine, newLine := skipped+1, skipped+1
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
	return out.String(), added, deleted, exact
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
