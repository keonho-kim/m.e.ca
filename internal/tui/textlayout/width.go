// Purpose: Measure terminal columns with a printable-ASCII fast path and Unicode fallback.
// Pattern: Pure Function.
// Usage: Terminal layout and drawing share Width; grapheme iterators reuse their own Width result.
// Related: internal/tui/textlayout/wrap.go, internal/tui/markdown/layout.go, internal/tui/view/cells.go.
package textlayout

import "github.com/rivo/uniseg"

// Width matches uniseg.StringWidth, including zero-width controls and combining characters.
func Width(text string) int {
	for i := range len(text) {
		if text[i] < ' ' || text[i] >= 0x7f {
			return uniseg.StringWidth(text)
		}
	}
	return len(text)
}
