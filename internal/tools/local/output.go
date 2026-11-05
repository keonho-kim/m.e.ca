// Purpose: Select UTF-8-safe head and tail excerpts from sanitized command output.
// Pattern: Pure bounded display projection.
// Usage: Live and final command projections share the same excerpt selection.
// Related: internal/tools/local/command.go, internal/tools/local/executor.go.
package local

import (
	"unicode/utf8"
)

const outputExcerptBytes = 32 * 1024

func commandExcerpt(text string) string {
	if len(text) <= outputExcerptBytes {
		return text
	}
	head, tail := outputExcerptBytes/2, len(text)-outputExcerptBytes/2
	for head > 0 && !utf8.RuneStart(text[head]) {
		head--
	}
	for tail < len(text) && !utf8.RuneStart(text[tail]) {
		tail++
	}
	return text[:head] + "\n[output excerpt; inspect the captured log path for omitted text]\n" + text[tail:]
}
