// Purpose: Own model-ID completion and distinguish filling a field from confirming a connection.
// Pattern: Event-loop state transitions.
// Usage: Connection results open the field; menu input routes candidate keys before confirmation.
// Related: internal/tui/modelmenu/suggestions.go, internal/tui/session/menu_keys.go, internal/tui/view/candidates.go.
package session

import (
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/v-code/internal/tui/modelmenu"
	"github.com/keonho-kim/v-code/internal/tui/view"
)

func (a *app) openModelInput(models []string) {
	a.page = "manual"
	a.choices = nil
	a.modelNames = append([]string(nil), models...)
	a.menuInput.Set("")
	a.refreshModelHints()
}
func (a *app) refreshModelHints() {
	if a.page != "manual" {
		return
	}
	a.modelHintsOpen = true
	a.modelHints = modelmenu.Suggestions(a.modelNames, a.menuInput.String())
	a.modelHintIndex = 0
	a.status = ""
}
func (a *app) closeModelHints() {
	a.modelHintsOpen = false
	a.modelHints = nil
	a.modelHintIndex = 0
}
func (a *app) modelHintKey(e *tcell.EventKey) bool {
	if a.page != "manual" || a.paste || a.menuInput.String() == ":q" {
		return false
	}
	if !a.modelHintsOpen {
		if e.Key() != tcell.KeyUp && e.Key() != tcell.KeyDown {
			return false
		}
		a.refreshModelHints()
	}
	switch e.Key() {
	case tcell.KeyUp:
		a.modelHintIndex = max(0, a.modelHintIndex-1)
	case tcell.KeyDown:
		a.modelHintIndex = min(max(0, len(a.modelHints)-1), a.modelHintIndex+1)
	case tcell.KeyEscape:
		a.closeModelHints()
	case tcell.KeyEnter, tcell.KeyTab:
		if len(a.modelHints) == 0 {
			return e.Key() == tcell.KeyTab
		}
		a.menuInput.Set(a.modelHints[a.modelHintIndex])
		a.closeModelHints()
		a.status = a.language.Text("모델명 입력됨 · Enter로 연결")
	default:
		return false
	}
	return true
}
func (a *app) modelCandidates() view.Candidates {
	labels := make([]string, len(a.modelHints))
	for i, name := range a.modelHints {
		labels[i] = a.opts.Vault.Redact(name)
	}
	return view.Candidates{Language: a.language, Open: a.modelHintsOpen, Labels: labels, Selected: a.modelHintIndex}
}
