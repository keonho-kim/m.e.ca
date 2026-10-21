// Purpose: Own classic selection weight and dark-text blue/red emphasis for file changes.
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
	switch change {
	case '+':
		return tcell.StyleDefault.Foreground(tcell.NewHexColor(0x24476b)).Background(tcell.NewHexColor(0xdbe5f0))
	case '-':
		return tcell.StyleDefault.Foreground(tcell.NewHexColor(0x702c35)).Background(tcell.NewHexColor(0xf0dadd))
	default:
		return tcell.StyleDefault
	}
}
