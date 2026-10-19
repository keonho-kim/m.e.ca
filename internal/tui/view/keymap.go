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
	help := Shortcut{}
	for _, shortcut := range orderedShortcuts(shortcuts) {
		if shortcut.Key == "F1" {
			help = shortcut
			continue
		}
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
		line := rows[last]
		measure := func(values []Shortcut) int {
			n := 0
			for i, s := range values {
				if i > 0 {
					n += 2
				}
				n += uniseg.StringWidth(s.Key + " " + s.Label)
			}
			return n
		}
		for help.Key != "" && len(line) > 0 && measure(append(slices.Clone(line), help)) > width && shortcutRank(line[len(line)-1].Key) > 1 {
			line = line[:len(line)-1]
		}
		if help.Key != "" && measure(append(slices.Clone(line), help)) <= width {
			line = append(line, help)
		} else if measure(line)+3 <= width {
			line = append(line, Shortcut{Key: "…"})
		}
		rows[last] = line
	}
	return rows
}

func shortcutRank(key string) int {
	if strings.HasPrefix(key, "^C") {
		return 0
	}
	ranks := []string{"^C", "Esc", "Enter", "Tab", "Backspace/Del", "^U", "Shift+Enter", "Ctrl/Cmd+↑↓", "Shift+Tab", "Space", "←→", "→", "←", "↑↓", "PgUp/Dn", "PgUp", "Home/End"}
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
