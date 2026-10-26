// Purpose: Describe only the model form's currently executable navigation, editing and save actions.
// Pattern: Presentation projection.
// Usage: screenShortcuts delegates the single-screen model settings hints here.
// Related: internal/tui/session/model_form_keys.go, internal/tui/view/keymap.go.
package session

import (
	"github.com/keonho-kim/meca/internal/commands"
	modelsettings "github.com/keonho-kim/meca/internal/tui/settings/model"
	"github.com/keonho-kim/meca/internal/tui/view"
	"strings"
)

func (a *app) modelSettingsShortcuts() []view.Shortcut {
	keys := []view.Shortcut{{Key: "^C", Label: "종료"}}
	f := a.modelForm
	if a.menuInput.String() == ":q" {
		return append(keys, view.Shortcut{Key: "Enter", Label: "복귀"}, view.Shortcut{Key: "Esc", Label: "취소"})
	}
	if command, err := commands.Parse(a.menuInput.String()); err == nil && (command.Kind == commands.Exit || command.Kind == commands.NewSession) {
		return append(keys, view.Shortcut{Key: "Enter", Label: a.commandEnterLabel(command.Kind)}, view.Shortcut{Key: "Esc", Label: "취소"})
	}
	if f != nil && f.Confirm {
		return settingsConfirmationShortcuts()
	}
	if a.worker.active() && (f == nil || !f.Catalog || !f.Editing || f.Row != modelsettings.Name) {
		if !a.worker.agent() && !a.worker.stopping() {
			keys = append(keys, view.Shortcut{Key: "Enter", Label: "Review save"})
		}
		return append(keys, view.Shortcut{Key: "Esc", Label: "연결 취소"})
	}
	if f != nil && f.Editing {
		move, tab := "이동", "Next setting"
		escape := "Finish editing"
		if a.worker.active() {
			escape = "조회 취소"
		}
		if f.Row == modelsettings.Name && len(f.Hints) > 0 {
			move, tab = "선택", "입력"
		}
		return append(keys, view.Shortcut{Key: "↑↓", Label: move}, view.Shortcut{Key: "Tab", Label: tab}, view.Shortcut{Key: "Enter", Label: "Review save"}, view.Shortcut{Key: "^U", Label: "Clear field"}, view.Shortcut{Key: "Esc", Label: escape})
	}
	keys = append(keys, view.Shortcut{Key: "↑↓", Label: "이동"}, view.Shortcut{Key: "Enter", Label: "Review save"}, view.Shortcut{Key: "Esc", Label: "닫기"})
	if f == nil {
		return keys
	}
	if f.Details {
		keys = append(keys, view.Shortcut{Key: "PgUp/Dn", Label: "모델 정보"})
	}
	label, previous := "", false
	switch f.Row {
	case modelsettings.Endpoint, modelsettings.Credential, modelsettings.Name:
		label = "입력"
	case modelsettings.Provider, modelsettings.Destination:
		label, previous = "Next value", true
	case modelsettings.Thinking, modelsettings.Effort:
		if f.Prepared == nil && strings.TrimSpace(f.Name.String()) != "" {
			label = "Inspect model"
			break
		}
		options := f.Options()
		count := len(options.Levels)
		if f.Row == modelsettings.Thinking {
			count = 0
			if options.Enabled() {
				count++
			}
			if options.Off {
				count++
			}
		}
		if count > 1 {
			label, previous = "Next value", true
		} else if count == 1 && f.Row == modelsettings.Effort {
			label = "선택"
		}
	case modelsettings.Information:
		label = "열기"
		if f.Details {
			label = "닫기"
		}
	case modelsettings.Diagnostic:
		next := f.Selection()
		if a.client != nil && next.Connection.ID == a.selection.Connection.ID && next.Model == a.selection.Model && !f.ControlsChanged && !f.KeyChanged {
			label = "실행"
		}
	}
	if label != "" {
		keys = append(keys, view.Shortcut{Key: "→", Label: label})
	}
	if previous {
		keys = append(keys, view.Shortcut{Key: "←", Label: "Previous value"})
	}
	return keys
}
