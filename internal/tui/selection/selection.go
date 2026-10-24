// Purpose: Select visible graphemes and preserve original line boundaries when copying.
// Pattern: Presentation state with immutable screen snapshot.
// Usage: Session mouse handlers begin/move/release; rendering overlays and copy reads Text.
// Related: internal/tui/session/text_selection.go, internal/tui/view/cells.go.
package selection

import (
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/meca/internal/tui/textlayout"
	"strings"
)

type cell struct {
	text         string
	style        tcell.Style
	width, owner int
}
type point struct{ x, y int }
type Selection struct {
	sources                    map[int]textlayout.Fragment
	cells                      [][]cell
	width, height, top, bottom int
	start, end                 point
	dragging, selected         bool
}

func (s *Selection) Sources(sources map[int]textlayout.Fragment) {
	s.sources = make(map[int]textlayout.Fragment, len(sources))
	for row, source := range sources {
		s.sources[row] = source
	}
}

func (s *Selection) Bounds() (int, int, int, int, bool) {
	if !s.selected {
		return 0, 0, 0, 0, false
	}
	a, b := s.bounds()
	return a.x, a.y, b.x, b.y, true
}

func (s *Selection) Clear()         { *s = Selection{} }
func (s *Selection) Dragging() bool { return s.dragging }
func (s *Selection) Active() bool   { return s.selected }
func (s *Selection) Begin(screen tcell.Screen, x, y, top, bottom int) {
	s.Clear()
	w, h := screen.Size()
	if w <= 0 || h <= 0 || w > 4096 || h > 4096 || w*h > 1024*1024 {
		return
	}
	top = max(0, top)
	bottom = min(h-1, bottom)
	if y < top || y > bottom || x < 0 || x >= w {
		return
	}
	s.width, s.height, s.top, s.bottom = w, h, top, bottom
	s.cells = make([][]cell, bottom-top+1)
	for row := top; row <= bottom; row++ {
		cells := make([]cell, w)
		for col := 0; col < w; {
			text, style, width := screen.Get(col, row)
			width = max(1, width)
			if text == "" {
				text = " "
			}
			cells[col] = cell{text, style, width, col}
			for n := col + 1; n < min(w, col+width); n++ {
				cells[n] = cell{owner: col}
			}
			col += width
		}
		s.cells[row-top] = cells
	}
	s.start = point{x, y}
	s.end = s.start
	s.dragging = true
}
func (s *Selection) Move(x, y int) {
	if !s.dragging {
		return
	}
	s.end = point{max(0, min(s.width-1, x)), max(s.top, min(s.bottom, y))}
	s.selected = s.start != s.end
}
func (s *Selection) Release() {
	s.dragging = false
	if !s.selected {
		s.Clear()
	}
}
func (s *Selection) bounds() (point, point) {
	a, b := s.start, s.end
	if a.y > b.y || a.y == b.y && a.x > b.x {
		a, b = b, a
	}
	a.x = s.cells[a.y-s.top][a.x].owner
	b.x = s.cells[b.y-s.top][b.x].owner
	return a, b
}
func (s *Selection) Text() string {
	if !s.selected {
		return ""
	}
	a, b := s.bounds()
	var result strings.Builder
	for row := a.y; row <= b.y; row++ {
		left, right := 0, s.width-1
		if row == a.y {
			left = a.x
		}
		if row == b.y {
			right = b.x
		}
		if source, ok := s.sources[row]; ok {
			start, end := source.Row.Offset(source.Source, left, false), source.Row.Offset(source.Source, right, true)
			result.WriteString(source.Source[start:end])
		} else {
			var line strings.Builder
			for x := left; x <= right; x++ {
				line.WriteString(s.cells[row-s.top][x].text)
			}
			result.WriteString(strings.TrimRight(line.String(), " "))
		}
		current, currentOK := s.sources[row]
		next, nextOK := s.sources[row+1]
		soft := currentOK && nextOK && current.Row.Soft && current.Source == next.Source && current.Row.End == next.Row.Start
		if row < b.y && !soft {
			result.WriteByte('\n')
		}
	}
	return result.String()
}
func (s *Selection) Draw(screen tcell.Screen) {
	if !s.selected {
		return
	}
	w, h := screen.Size()
	if w != s.width || h != s.height {
		s.Clear()
		return
	}
	a, b := s.bounds()
	for row := s.top; row <= s.bottom; row++ {
		for x, c := range s.cells[row-s.top] {
			if c.width == 0 {
				continue
			}
			selected := s.selected && (row > a.y || row == a.y && x >= a.x) && (row < b.y || row == b.y && x <= b.x)
			if selected {
				screen.Put(x, row, c.text, c.style.Reverse(true))
			}
		}
	}
	screen.HideCursor()
}
