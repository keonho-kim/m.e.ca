// Purpose: Project the staged interface language into the shared settings screen.
// Pattern: Read-only presentation snapshot.
// Usage: Session rendering calls this while !language owns the screen.
// Related: internal/tui/session/language.go, internal/tui/view/settings.go.
package session

import (
	"github.com/keonho-kim/meca/internal/tui/view"
)

func (a *app) renderLanguage(w, h int) {
	a.headerTitle = a.language.Text("Interface language")
	labels := make([]string, len(languageChoices))
	for i, choice := range languageChoices {
		labels[i] = a.language.Text(choice.label)
	}
	view.Settings(a.screen, view.SettingsDisplay{
		Title: a.headerTitle, Language: a.language,
		Activity: a.activity(), Usage: a.chat.displayTokens(), Status: a.status,
		Confirm: a.languageConfirm, Command: a.menuInput.String(),
		Rows: []view.SettingRow{{Label: a.language.Text("Interface language"), Values: labels, Selected: a.choice}},
	}, w, h)
}
