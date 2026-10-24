// Purpose: Project command suggestions into sanitized candidate labels.
// Pattern: Presentation projection.
// Usage: Chat layout supplies candidates to view.Chat.
// Related: internal/tui/session/hints.go, internal/tui/session/keys.go, internal/tui/view/candidates.go.
package session

import "github.com/keonho-kim/meca/internal/tui/view"

func (a *app) candidates() view.Candidates {
	if !a.commandOpen {
		return view.Candidates{}
	}
	labels := make([]string, len(a.commandHints))
	for i, suggestion := range a.commandHints {
		labels[i] = suggestion.Name + "  " + suggestion.Description
	}
	return view.Candidates{Language: a.language, Open: true, Labels: labels, Selected: a.commandIndex}
}
