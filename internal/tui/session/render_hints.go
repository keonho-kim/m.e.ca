// Purpose: Project command and file-reference suggestions into sanitized candidate labels.
// Pattern: Presentation projection.
// Usage: Chat layout supplies candidates to view.Chat.
// Related: internal/tui/session/hints.go, internal/tui/session/completion.go, internal/tui/view/candidates.go.
package session

import "github.com/keonho-kim/v-code/internal/tui/view"

func (a *app) candidates() view.Candidates {
	if !a.commandOpen && !a.refOpen {
		return view.Candidates{}
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
	return view.Candidates{Open: true, Labels: labels, Selected: index}
}
