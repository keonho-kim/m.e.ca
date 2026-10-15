// Purpose: Own classic selection weight and muted addition/deletion emphasis in approval previews.
// Pattern: Pure Function.
// Usage: Used by candidate lists, menus, file columns, conversation selection and diff approval.
// Related: internal/tui/session/render_hints.go, internal/tui/session/render.go, internal/tui/session/render_files.go, internal/tui/startup/resume.go.
package view

import "github.com/gdamore/tcell/v3"

func SelectionStyle(selected bool) tcell.Style {
	if selected {
		return tcell.StyleDefault.Bold(true)
	}
	return tcell.StyleDefault.Dim(true)
}

func ChangeStyle(change byte) tcell.Style {
	style := tcell.StyleDefault.Foreground(tcell.NewHexColor(0xd8d8d2))
	switch change {
	case '+':
		return style.Background(tcell.NewHexColor(0x20372c))
	case '-':
		return style.Background(tcell.NewHexColor(0x422a2d))
	default:
		return tcell.StyleDefault
	}
}
