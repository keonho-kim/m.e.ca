// Purpose: Wrap printable ASCII by byte offsets while preserving the Unicode layout contract.
// Pattern: Proven narrow fast path with the existing grapheme fallback.
// Usage: RowsFrom uses asciiRows only when every paragraph byte is one terminal cell.
// Related: internal/tui/textlayout/wrap.go, internal/tui/textlayout/width.go.
package textlayout

import (
	"strings"
)

func printableASCII(text string) bool {
	for i := range len(text) {
		if text[i] < ' ' || text[i] >= 0x7f {
			return false
		}
	}
	return true
}

func asciiRows(text string, base, width int, prefix string, resumed bool, yield func(Row) bool) bool {
	indent := text[:len(text)-len(strings.TrimLeft(text, " "))]
	if resumed {
		indent = prefix
	}
	indent = indent[:min(len(indent), max(0, width-2))]
	at := 0
	for {
		end := min(len(text), at+width-len(prefix))
		row := Row{Text: text[at:end], Start: base + at, End: base + end, Prefix: prefix, Soft: end < len(text)}
		if prefix != "" {
			row.Text = prefix + row.Text
		}
		if !yield(row) {
			return false
		}
		if end == len(text) {
			return true
		}
		at, prefix = end, indent
	}
}
