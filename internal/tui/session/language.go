// Purpose: Stage and confirm the terminal language preference before persisting and applying it.
// Pattern: Event-loop settings use case.
// Usage: !language opens settings; arrow keys select and y confirms saving after Enter.
// Related: internal/commands/settings/language.go, internal/tui/session/confirmation.go, internal/tui/session/render_language.go.
package session

import (
	"strings"

	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/meca/internal/commands"
	settingcmd "github.com/keonho-kim/meca/internal/commands/settings"
	"github.com/keonho-kim/meca/internal/config"
	"github.com/keonho-kim/meca/internal/tui/input"
	"github.com/keonho-kim/meca/internal/tui/locale"
	"github.com/keonho-kim/meca/internal/tui/view"
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
	a.languageConfirm = false
	for i, choice := range languageChoices {
		if choice.value == a.opts.Config.Language {
			a.choice = i
		}
	}
	a.menuInput.Set("")
}

func (a *app) languageKey(e *tcell.EventKey) {
	if a.languageConfirm {
		switch confirmationKey(e) {
		case confirmEdit:
			a.languageConfirm = false
		case confirmAccept:
			a.languageConfirm = false
			a.saveLanguage()
		case confirmReject:
			a.languageConfirm = false
			a.page, a.status = "chat", ""
		}
		return
	}
	if a.menuInput.String() != "" || e.Key() == tcell.KeyRune && (strings.HasPrefix(e.Str(), ":") || strings.HasPrefix(e.Str(), "!")) {
		switch e.Key() {
		case tcell.KeyEscape:
			a.menuInput.Set("")
		case tcell.KeyEnter:
			if a.menuInput.String() == ":q" {
				a.page = "chat"
				a.menuInput.Set("")
			}
		default:
			input.EditKey(&a.menuInput, e)
		}
		return
	}
	switch e.Key() {
	case tcell.KeyRight, tcell.KeyLeft:
		delta := 1
		if e.Key() == tcell.KeyLeft {
			delta = -1
		}
		a.choice = (a.choice + delta + len(languageChoices)) % len(languageChoices)
	case tcell.KeyEnter:
		a.languageConfirm = true
	case tcell.KeyEscape:
		a.page, a.status = "chat", ""
	}
}

func (a *app) languageShortcuts() []view.Shortcut {
	if a.languageConfirm {
		return settingsConfirmationShortcuts()
	}
	keys := []view.Shortcut{{Key: "^C", Label: "종료"}}
	if a.menuInput.String() != "" {
		if command, err := commands.Parse(a.menuInput.String()); err == nil && (command.Kind == commands.Quit || command.Kind == commands.Exit || command.Kind == commands.NewSession) {
			keys = append(keys, view.Shortcut{Key: "Enter", Label: a.commandEnterLabel(command.Kind)})
		}
		return append(keys, view.Shortcut{Key: "Esc", Label: "취소"}, view.Shortcut{Key: "^U", Label: "앞부분 지우기"})
	}
	return append(keys, view.Shortcut{Key: "→", Label: "Next value"}, view.Shortcut{Key: "←", Label: "Previous value"}, view.Shortcut{Key: "Enter", Label: "Review save"}, view.Shortcut{Key: "Esc", Label: "닫기"})
}

func (a *app) saveLanguage() {
	if a.worker.active() {
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
