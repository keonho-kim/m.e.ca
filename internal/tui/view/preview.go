// Purpose: Preserve added/removed line classification while wrapping unified approval previews.
// Pattern: Pure presentation projection.
// Usage: The approval renderer supplies frozen preview text and its explicit diff marker.
// Related: internal/tui/session/render.go, internal/tui/viewport/viewport.go, internal/agent/tools.go.
package view

import (
	"strings"

	"github.com/keonho-kim/v-code/internal/tui/viewport"
)

type PreviewLine struct {
	Text   string
	Change byte
}

func PreviewLines(preview string, width int, diff bool) []PreviewLine {
	var rows []PreviewLine
	inHunk := false
	for _, line := range strings.Split(preview, "\n") {
		if diff && strings.HasPrefix(line, "@@ -") {
			inHunk = true
		}
		var change byte
		if inHunk && (strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-")) {
			change = line[0]
		}
		for _, wrapped := range viewport.Wrap(line, width) {
			rows = append(rows, PreviewLine{Text: wrapped, Change: change})
		}
	}
	return rows
}
