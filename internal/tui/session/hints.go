// Purpose: Own command suggestion state and keyboard selection for input candidates.
// Pattern: Presentation state transitions.
// Usage: Refreshed after input edits; consumes selection keys before editor or submission handling.
// Related: internal/commands/catalog.go, internal/tui/session/keys.go, internal/tui/session/render_hints.go.
package session

import (
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/meca/internal/commands"
	"strings"
	"unicode"
)

func (a *app) closeCommandHints() {
	a.commandOpen = false
	a.commandHints = nil
	a.commandIndex = 0
}
func (a *app) refreshHints() {
	a.closeCommandHints()
	if a.paste {
		return
	}
	prefix := strings.TrimLeftFunc(a.editor.String(), unicode.IsSpace)
	if a.page == "chat" && a.editor.Cursor == a.editor.Len() && strings.HasPrefix(prefix, "!") && strings.IndexFunc(prefix, unicode.IsSpace) < 0 {
		a.commandOpen = true
		a.commandHints = commands.Suggestions(prefix, a.language.Text)
		return
	}
}
func (a *app) hintKey(e *tcell.EventKey) bool {
	if a.paste || e.Key() == tcell.KeyEnter && e.Modifiers()&tcell.ModShift != 0 {
		return false
	}
	if a.commandOpen {
		switch e.Key() {
		case tcell.KeyUp:
			a.commandIndex = max(0, a.commandIndex-1)
		case tcell.KeyDown:
			a.commandIndex = min(max(0, len(a.commandHints)-1), a.commandIndex+1)
		case tcell.KeyEscape:
			a.closeCommandHints()
		case tcell.KeyEnter, tcell.KeyTab:
			if len(a.commandHints) > 0 {
				a.editor.Set(a.commandHints[a.commandIndex].Name)
				a.closeCommandHints()
				if e.Key() == tcell.KeyEnter {
					a.submit()
				}
			}
		default:
			return false
		}
		return true
	}

	return false
}
