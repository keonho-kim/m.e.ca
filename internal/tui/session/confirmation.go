// Purpose: Interpret explicit y/N/Esc confirmation choices and project settings confirmation hints.
// Pattern: Simple Module.
// Usage: Settings and file submission share confirmation keys; settings also share save labels.
// Related: internal/tui/session/model_form_keys.go, internal/tui/session/language.go, internal/tui/session/file_submission.go.
package session

import (
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/meca/internal/tui/view"
)

type confirmationDecision uint8

const (
	confirmUndecided confirmationDecision = iota
	confirmAccept
	confirmReject
	confirmEdit
)

func confirmationKey(e *tcell.EventKey) confirmationDecision {
	if e.Key() == tcell.KeyEscape {
		return confirmEdit
	}
	if e.Key() == tcell.KeyRune && e.Modifiers()&^(tcell.ModShift) == 0 {
		switch e.Str() {
		case "y", "Y":
			return confirmAccept
		case "N":
			return confirmReject
		}
	}
	return confirmUndecided
}

func settingsConfirmationShortcuts() []view.Shortcut {
	return []view.Shortcut{{Key: "^C", Label: "종료"}, {Key: "y", Label: "Save"}, {Key: "N", Label: "Discard"}, {Key: "Esc", Label: "Keep editing"}}
}
