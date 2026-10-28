// Purpose: Own Markdown preview detection, display-mode transitions and cached styled windows.
// Pattern: Presentation adapter over the existing Markdown viewport.
// Usage: Preview routes Markdown files to these projections; viewer keys toggle their format.
// Related: internal/tui/browser/preview.go, internal/tui/viewport/block_layout.go.
package browser

import (
	"path/filepath"
	"strings"

	"github.com/keonho-kim/meca/internal/tui/viewport"
)

func (p *Preview) IsMarkdown() bool {
	switch strings.ToLower(filepath.Ext(p.Path)) {
	case ".md", ".markdown":
		return true
	default:
		return false
	}
}

func (p *Preview) Rendered() bool { return p.IsMarkdown() && !p.Raw }

func (p *Preview) ToggleFormat() {
	if !p.IsMarkdown() {
		return
	}
	p.Raw = !p.Raw
	p.Offset = 0
	p.formatted = viewport.Block{}
	p.index, p.content = nil, ""
}

func (p *Preview) markdownCount(width int) int {
	p.formatted.Text, p.formatted.Markdown = p.Content, true
	count := p.formatted.Count(max(1, width))
	if p.Truncated {
		count++
	}
	return count
}

func (p *Preview) markdownWindow(width, start, count int) []viewport.Line {
	width = max(1, width)
	total := p.formatted.Count(width)
	rows := p.formatted.Window(width, start, count)
	if p.Truncated && len(rows) < count && start+len(rows) >= total {
		rows = append(rows, viewport.Line{Text: previewTruncated})
	}
	return rows
}
