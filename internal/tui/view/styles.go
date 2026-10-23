// Purpose: Own classic selection weight and dark blue/red backgrounds with bright text for file changes.
// Pattern: Pure Function.
// Usage: Candidate lists and menus share selection weights; approvals and tool results share diff colors.
// Related: internal/tui/view/chat.go, internal/tui/view/menu.go, internal/tui/startup/resume.go.
package view

import "github.com/gdamore/tcell/v3"

func SelectionStyle(selected bool) tcell.Style {
	if selected {
		return tcell.StyleDefault.Bold(true)
	}
	return tcell.StyleDefault.Dim(true)
}

func ChangeStyle(change byte) tcell.Style {
	style := tcell.StyleDefault.Foreground(tcell.NewHexColor(0xe6edf3))
	switch change {
	case '+':
		return style.Background(tcell.NewHexColor(0x123b63))
	case '-':
		return style.Background(tcell.NewHexColor(0x6b2230))
	default:
		return tcell.StyleDefault
	}
}
