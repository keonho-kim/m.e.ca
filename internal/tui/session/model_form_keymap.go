// Purpose: Describe only the model form's currently executable navigation, editing and save actions.
// Pattern: Presentation projection.
// Usage: screenShortcuts delegates the single-screen model settings hints here.
// Related: internal/tui/session/model_form_keys.go, internal/tui/view/keymap.go.
package session

import (
	"github.com/keonho-kim/meca/internal/commands"
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
	if f != nil && f.confirm {
		return settingsConfirmationShortcuts()
	}
	if a.worker.active() && (f == nil || !f.catalog || !f.editing || f.row != modelName) {
		if !a.worker.agent() && !a.worker.stopping() {
			keys = append(keys, view.Shortcut{Key: "Enter", Label: "Review save"})
		}
		return append(keys, view.Shortcut{Key: "Esc", Label: "연결 취소"})
	}
	if f != nil && f.editing {
		move, tab := "이동", "Next setting"
		escape := "Finish editing"
		if a.worker.active() {
			escape = "조회 취소"
		}
		if f.row == modelName && len(f.hints) > 0 {
			move, tab = "선택", "입력"
		}
		return append(keys, view.Shortcut{Key: "↑↓", Label: move}, view.Shortcut{Key: "Tab", Label: tab}, view.Shortcut{Key: "Enter", Label: "Review save"}, view.Shortcut{Key: "^U", Label: "Clear field"}, view.Shortcut{Key: "Esc", Label: escape})
	}
	keys = append(keys, view.Shortcut{Key: "↑↓", Label: "이동"}, view.Shortcut{Key: "Enter", Label: "Review save"}, view.Shortcut{Key: "Esc", Label: "닫기"})
	if f == nil {
		return keys
	}
	if f.details {
		keys = append(keys, view.Shortcut{Key: "PgUp/Dn", Label: "모델 정보"})
	}
	label, previous := "", false
	switch f.row {
	case modelEndpoint, modelCredential, modelName:
		label = "입력"
	case modelProvider, modelDestination:
		label, previous = "Next value", true
	case modelThinking, modelEffort:
		if f.prepared == nil && strings.TrimSpace(f.name.String()) != "" {
			label = "Inspect model"
			break
		}
		options := f.options()
		count := len(options.Levels)
		if f.row == modelThinking {
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
		} else if count == 1 && f.row == modelEffort {
			label = "선택"
		}
	case modelInformation:
		label = "열기"
		if f.details {
			label = "닫기"
		}
	case modelDiagnostic:
		next := f.selection()
		if a.client != nil && next.Connection.ID == a.selection.Connection.ID && next.Model == a.selection.Model && !f.controlsChanged && !f.keyChanged {
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
