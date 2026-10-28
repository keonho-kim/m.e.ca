// Purpose: Detect Markdown paths and project their prepared styled windows.
// Pattern: Presentation adapter over the existing Markdown viewport.
// Usage: Worker-owned Layout delegates Markdown parsing and cached windows here.
// Related: internal/tui/preview/raw.go, internal/tui/viewport/block_layout.go.
package preview

import (
	"path/filepath"
	"strings"

	"github.com/keonho-kim/meca/internal/tui/viewport"
)

func markdownPath(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".markdown":
		return true
	default:
		return false
	}
}

func (p *Layout) Rendered() bool { return markdownPath(p.Path) && !p.Raw }

func (p *Layout) markdownCount(width int) int {
	p.formatted.Text, p.formatted.Markdown = p.Content, true
	count := p.formatted.Count(max(1, width))
	if p.Truncated {
		count++
	}
	return count
}

func (p *Layout) markdownWindow(width, start, count int) []viewport.Line {
	width = max(1, width)
	total := p.formatted.Count(width)
	rows := p.formatted.Window(width, start, count)
	if p.Truncated && len(rows) < count && start+len(rows) >= total {
		rows = append(rows, viewport.Line{Text: previewTruncated})
	}
	return rows
}
