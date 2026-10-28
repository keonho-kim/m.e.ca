// Purpose: Own visible preview mode and position without performing layout on the terminal loop.
// Pattern: Presentation state receiving a prepared immutable-source layout.
// Usage: Session renders cached windows and submits Snapshot values to its bounded layout worker.
// Related: internal/tui/preview/build.go, internal/tui/session/preview_layout.go.
package preview

import (
	"github.com/keonho-kim/meca/internal/tui/viewport"
)

type State struct {
	Path, Content, Error     string
	Full, Loading, Truncated bool
	Raw                      bool
	Offset                   int
	layout                   *Layout
	key                      Snapshot
}

func (p *State) IsMarkdown() bool {
	return markdownPath(p.Path)
}

func (p *State) ToggleFormat() {
	if p.IsMarkdown() {
		p.Raw, p.Offset, p.layout = !p.Raw, 0, nil
	}
}

func (p *State) Snapshot(width int) Snapshot {
	return Snapshot{Path: p.Path, Content: p.Content, Raw: p.Raw, Truncated: p.Truncated, Width: max(1, width)}
}

func (p *State) NeedsLayout(width int) bool {
	return !p.Loading && p.Error == "" && p.Content != "" && (p.layout == nil || p.key != p.Snapshot(width))
}

func (p *State) Apply(key Snapshot, layout *Layout) bool {
	if key != p.Snapshot(key.Width) {
		return false
	}
	p.key, p.layout = key, layout
	return true
}

func (p *State) ReleaseLayout() { p.layout = nil }

func (p *State) placeholder(width int) []string {
	switch {
	case p.Loading:
		return []string{"Reading file…"}
	case p.Error != "":
		return viewport.Wrap(p.Error, max(1, width))
	case p.Content == "":
		p.layout = nil
		return []string{"(Empty file)"}
	case p.NeedsLayout(width):
		return []string{"Preparing file display…"}
	}
	return nil
}

func (p *State) Count(width int) int {
	if lines := p.placeholder(width); lines != nil {
		return len(lines)
	}
	return p.layout.Count(width)
}

func (p *State) Window(width, start, count int) []viewport.Line {
	if lines := p.placeholder(width); lines != nil {
		var rows []viewport.Line
		start = min(max(0, start), len(lines))
		for _, text := range lines[start:min(len(lines), start+max(0, count))] {
			rows = append(rows, viewport.Line{Text: text})
		}
		return rows
	}
	return p.layout.Window(width, start, count)
}
