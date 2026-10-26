// Purpose: Place primary actions before supporting hints in one or two bounded footer rows.
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

// MaxKeymapRows applies to both session screens and the conversation picker.
const MaxKeymapRows = 2

var primaryKeys = []string{"^C", "Esc", "Enter", "Ctrl/Cmd+Backspace", "Tab", "Shift+↑↓/←→", "Shift+↑/←", "↑↓", "←→", "←", "→", "Space", "Shift+Tab", "!", ":q"}
var supportingKeys = []string{"Backspace/Del", "^U", "Shift+Enter", "Ctrl/Cmd+↑↓", "^O", "PgUp/Dn", "PgUp", "Home/End"}

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
	maxRows = min(maxRows, MaxKeymapRows)
	ordered := orderedShortcuts(shortcuts)
	needed := -2
	for _, shortcut := range ordered {
		needed += uniseg.StringWidth(shortcut.Key) + 1 + uniseg.StringWidth(shortcut.Label) + 2
	}
	separate := maxRows > 1 && needed > width
	rows := [][]Shortcut{{}}
	cells := 0
	for _, shortcut := range ordered {
		count := uniseg.StringWidth(shortcut.Key) + 1 + uniseg.StringWidth(shortcut.Label)
		if count > width {
			continue
		}
		if cells > 0 && (cells+count > width || separate && len(rows) == 1 && supportingShortcut(shortcut.Key)) {
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
	if n := slices.Index(primaryKeys, key); n >= 0 {
		return n
	}
	if n := slices.Index(supportingKeys, key); n >= 0 {
		return len(primaryKeys) + n
	}
	return len(primaryKeys) + len(supportingKeys)
}

func supportingShortcut(key string) bool {
	return slices.Contains(supportingKeys, key)
}

func orderedShortcuts(shortcuts []Shortcut) []Shortcut {
	ordered := slices.Clone(shortcuts)
	slices.SortStableFunc(ordered, func(a, b Shortcut) int { return shortcutRank(a.Key) - shortcutRank(b.Key) })
	return ordered
}
