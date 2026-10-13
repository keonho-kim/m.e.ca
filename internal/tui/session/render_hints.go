// Purpose: Render command and file candidates immediately below the input with bounded height.
// Pattern: Presentation renderer.
// Usage: Called by the chat renderer; returns the first row after the candidate list.
// Related: internal/tui/session/hints.go, internal/tui/session/completion.go, internal/tui/view/styles.go, internal/tui/session/render.go.
package session

import (
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/v-code/internal/tui/view"
)

func (a *app) renderHints(y, width, height int) int {
	if !a.commandOpen && !a.refOpen {
		return y
	}
	var labels []string
	index := a.refIndex
	if a.commandOpen {
		index = a.commandIndex
		for _, suggestion := range a.commandHints {
			labels = append(labels, suggestion.Name+"  "+suggestion.Description)
		}
	} else {
		for _, ref := range a.refs {
			labels = append(labels, a.opts.Vault.Redact(ref.Path))
		}
	}
	count := min(6, max(0, height-y-2))
	if count == 0 {
		return y
	}
	if len(labels) == 0 {
		view.Text(a.screen, 0, y, width, "No matches", tcell.StyleDefault.Dim(true))
		return y + 1
	}
	start := max(0, index-count+1)
	for i := start; i < min(len(labels), start+count); i++ {
		view.Text(a.screen, 0, y, width, labels[i], view.SelectionStyle(i == index))
		y++
	}
	return y
}
