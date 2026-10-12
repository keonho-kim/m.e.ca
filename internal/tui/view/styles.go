// Purpose: Emphasize selection using text weight without changing terminal foreground or background colors.
// Pattern: Pure Function.
// Usage: Shared by candidate lists, menus, file columns and conversation selection.
// Related: internal/tui/session/render_hints.go, internal/tui/session/render.go, internal/tui/session/render_files.go, internal/tui/startup/resume.go.
package view

import "github.com/gdamore/tcell/v2"

func SelectionStyle(selected bool) tcell.Style {
	if selected {
		return tcell.StyleDefault.Bold(true)
	}
	return tcell.StyleDefault.Dim(true)
}
