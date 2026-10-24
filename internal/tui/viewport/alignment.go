// Purpose: Center projected rows while keeping copy coordinates connected to their original text.
// Pattern: Pure layout transformation.
// Usage: Centered viewport blocks apply centerLines after wrapping at the current width.
// Related: internal/tui/viewport/viewport.go, internal/tui/textlayout/wrap.go.
package viewport

import (
	"strings"

	"github.com/keonho-kim/meca/internal/tui/markdown"
	"github.com/rivo/uniseg"
)

func centerLines(lines []Line, width int) {
	for i := range lines {
		line := &lines[i]
		if strings.TrimSpace(line.Text) == "" {
			continue
		}
		padding := strings.Repeat(" ", max(0, width-uniseg.StringWidth(line.Text))/2)
		line.Text = padding + line.Text
		line.Copy.Row.Prefix = padding + line.Copy.Row.Prefix
		if len(line.Spans) > 0 && padding != "" {
			line.Spans = append([]markdown.Span{{Text: padding}}, line.Spans...)
		}
	}
}
