// Purpose: Select, persist and immediately apply the global terminal language preference.
// Pattern: Event-loop settings use case.
// Usage: !language opens this menu; Enter saves a choice and refreshes localized history.
// Related: internal/commands/settings/language.go, internal/tui/locale/preference.go, internal/tui/session/history.go.
package session

import (
	settingcmd "github.com/keonho-kim/v-code/internal/commands/settings"
	"github.com/keonho-kim/v-code/internal/config"
	"github.com/keonho-kim/v-code/internal/tui/locale"
)

var languageChoices = []struct {
	value config.Language
	label string
}{
	{config.LanguageAuto, "System default"},
	{config.LanguageKorean, "한국어"},
	{config.LanguageEnglish, "English"},
}

func (a *app) openLanguage() {
	a.cancelDirectory()
	a.stopNotice, a.status = "", ""
	a.page, a.choice = "language", 0
	a.choices = nil
	for i, choice := range languageChoices {
		a.choices = append(a.choices, choice.label)
		if choice.value == a.opts.Config.Language {
			a.choice = i
		}
	}
	a.menuInput.Set("")
}

func (a *app) chooseLanguage() {
	if a.busy {
		a.status = a.language.Text("Stop the current task before changing settings")
		return
	}
	next, err := settingcmd.Language(a.opts.Config, languageChoices[a.choice].value, a.opts.SaveConfig)
	if err != nil {
		a.fail(err)
		return
	}
	a.opts.Config = next
	a.language = locale.Resolve(a.ctx, next.Language)
	a.clearSelection()
	a.chatProjection = chatProjection{}
	a.scrollbar = scrollbarState{}
	a.stopNotice, a.status = "", ""
	a.menuInput.Set("")
	a.page = "chat"
	a.fail(a.loadHistory())
}
