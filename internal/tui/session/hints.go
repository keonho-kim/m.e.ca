// Purpose: Own command suggestion state and keyboard selection for input candidates.
// Pattern: Presentation state transitions.
// Usage: Refreshed after input edits; consumes selection keys before editor or submission handling.
// Related: internal/commands/catalog.go, internal/tui/session/keys.go, internal/tui/session/render_hints.go.
package session

import (
	"strings"
	"unicode"

	"github.com/gdamore/tcell/v3"

	"github.com/keonho-kim/meca/internal/commands"
)

func (a *app) closeCommandHints() {
	a.compose.commandOpen = false
	a.compose.commandHints = nil
	a.compose.commandIndex = 0
}
func (a *app) refreshHints() {
	a.closeCommandHints()
	if a.paste || a.compose.queued.Edit != nil {
		return
	}
	prefix := strings.TrimLeftFunc(a.compose.editor.String(), unicode.IsSpace)
	if a.page == "chat" && a.compose.editor.Cursor == a.compose.editor.Len() && strings.HasPrefix(prefix, "!") && strings.IndexFunc(prefix, unicode.IsSpace) < 0 {
		a.compose.commandOpen = true
		a.compose.commandHints = commands.Suggestions(prefix, a.language.Text)
		return
	}
	if a.page == "chat" && a.compose.editor.Cursor == a.compose.editor.Len() {
		if query := skillCompletion(prefix); query != "" {
			a.requestSkillHints(query)
		}
	}
}
func (a *app) hintKey(e *tcell.EventKey) bool {
	if a.paste || e.Key() == tcell.KeyEnter && e.Modifiers()&tcell.ModShift != 0 {
		return false
	}
	if a.compose.commandOpen {
		switch e.Key() {
		case tcell.KeyUp:
			a.compose.commandIndex = max(0, a.compose.commandIndex-1)
		case tcell.KeyDown:
			a.compose.commandIndex = min(max(0, len(a.compose.commandHints)-1), a.compose.commandIndex+1)
		case tcell.KeyEscape:
			a.closeCommandHints()
		case tcell.KeyEnter, tcell.KeyTab:
			if len(a.compose.commandHints) > 0 {
				name := a.compose.commandHints[a.compose.commandIndex].Name
				if strings.HasPrefix(name, "#") {
					text := a.compose.editor.String()
					query := skillCompletion(text)
					a.compose.editor.Set(strings.TrimSuffix(text, query) + name + " ")
					a.closeCommandHints()
					return true
				}
				a.compose.editor.Set(name)
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
