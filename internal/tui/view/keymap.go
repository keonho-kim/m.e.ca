// Purpose: Measure and render bounded nano-style keyboard hints without owning screen state.
// Pattern: Presentation renderer with pure row layout.
// Usage: Session and startup supply key-label pairs and a reserved bottom row range.
// Related: internal/tui/session/keymap.go, internal/tui/startup/resume.go, internal/tui/view/cells.go.
package view

import (
	"github.com/gdamore/tcell/v3"
	"github.com/rivo/uniseg"
	"slices"
	"strings"
)

type Shortcut struct{ Key, Label string }

func KeymapRows(shortcuts []Shortcut, width, maxRows int) int {
	return len(keymapLines(shortcuts, width, maxRows))
}

func Keymap(screen tcell.Screen, shortcuts []Shortcut, y, width, maxRows int) {
	for row, shortcuts := range keymapLines(shortcuts, width, maxRows) {
		drawShortcuts(screen, shortcuts, 0, y+row, width)
	}
}

func drawShortcuts(screen tcell.Screen, shortcuts []Shortcut, x, y, end int) {
	for _, shortcut := range shortcuts {
		Text(screen, x, y, end-x, shortcut.Key, tcell.StyleDefault.Bold(true))
		x += uniseg.StringWidth(shortcut.Key) + 1
		if x < end {
			Text(screen, x, y, end-x, shortcut.Label, tcell.StyleDefault.Dim(true))
		}
		x += uniseg.StringWidth(shortcut.Label) + 2
	}
}

func keymapLines(shortcuts []Shortcut, width, maxRows int) [][]Shortcut {
	if width < 1 || maxRows < 1 || len(shortcuts) == 0 {
		return nil
	}
	rows := [][]Shortcut{{}}
	cells := 0
	for _, shortcut := range orderedShortcuts(shortcuts) {
		count := uniseg.StringWidth(shortcut.Key) + 1 + uniseg.StringWidth(shortcut.Label)
		if count > width {
			continue
		}
		if cells > 0 && cells+count > width {
			if len(rows) == maxRows {
				continue
			}
			rows = append(rows, []Shortcut{})
			cells = 0
		}
		rows[len(rows)-1] = append(rows[len(rows)-1], shortcut)
		cells += count + 2
	}
	if len(rows[0]) == 0 {
		return nil
	}
	count := 0
	for _, row := range rows {
		count += len(row)
	}
	if count < len(shortcuts) {
		last := len(rows) - 1
		used := 0
		for _, shortcut := range rows[last] {
			used += uniseg.StringWidth(shortcut.Key+" "+shortcut.Label) + 2
		}
		if used+1 <= width {
			rows[last] = append(rows[last], Shortcut{Key: "…"})
		}
	}
	return rows
}

func shortcutRank(key string) int {
	if strings.HasPrefix(key, "^C") {
		return 0
	}
	ranks := []string{"^C", "Esc", "Enter", "Tab", "^O", "Backspace/Del", "^U", "Shift+Enter", "Ctrl/Cmd+↑↓", "Shift+Tab", "Space", "←→", "→", "←", "↑↓", "PgUp/Dn", "PgUp", "Home/End"}
	if n := slices.Index(ranks, key); n >= 0 {
		return n
	}
	return len(ranks)
}

func orderedShortcuts(shortcuts []Shortcut) []Shortcut {
	ordered := slices.Clone(shortcuts)
	slices.SortStableFunc(ordered, func(a, b Shortcut) int { return shortcutRank(a.Key) - shortcutRank(b.Key) })
	return ordered
}
