// Purpose: Own classic selection weight and pastel addition/deletion emphasis in approval previews.
// Pattern: Pure Function.
// Usage: Used by candidate lists, menus, file columns, conversation selection and diff approval.
// Related: internal/tui/session/render_hints.go, internal/tui/session/render.go, internal/tui/session/render_files.go, internal/tui/startup/resume.go.
package view

import "github.com/gdamore/tcell/v2"

func SelectionStyle(selected bool) tcell.Style {
	if selected {
		return tcell.StyleDefault.Bold(true)
	}
	return tcell.StyleDefault.Dim(true)
}

func ChangeStyle(change byte) tcell.Style {
	style := tcell.StyleDefault.Foreground(tcell.NewHexColor(0x242424))
	switch change {
	case '+':
		return style.Background(tcell.NewHexColor(0xdcfce7))
	case '-':
		return style.Background(tcell.NewHexColor(0xfee2e2))
	default:
		return tcell.StyleDefault
	}
}
