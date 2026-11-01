// Purpose: Resolve the active interaction target once for input, key hints and screen rendering.
// Pattern: Pure focus projection from event-loop state.
// Usage: Keyboard routing, footer hints and rendering consume the same focusKind.
// Related: internal/tui/session/keys.go, internal/tui/session/keymap.go, internal/tui/session/render.go.
package session

type focusKind uint8

const (
	focusMenu focusKind = iota
	focusChat
	focusCommands
	focusQueue
	focusFileNavigation
	focusFileInput
	focusFilePreview
	focusFileConfirmation
	focusModel
	focusLanguage
	focusPermission
	focusApproval
)

func (a *app) focus() focusKind {
	switch a.page {
	case "chat":
		if a.compose.queued.Edit != nil {
			return focusQueue
		}
		if a.compose.commandOpen {
			return focusCommands
		}
		return focusChat
	case "files":
		if a.files.fileSendConfirm {
			return focusFileConfirmation
		}
		if a.files.filePreview != nil {
			return focusFilePreview
		}
		if a.files.browser.Input {
			return focusFileInput
		}
		return focusFileNavigation
	case "model":
		return focusModel
	case "language":
		return focusLanguage
	case "permission":
		return focusPermission
	case "approval":
		return focusApproval
	}
	return focusMenu
}

func (f focusKind) composer() bool {
	return f == focusChat || f == focusCommands || f == focusQueue || f == focusFileInput
}

func (f focusKind) page() string {
	switch f {
	case focusChat, focusCommands, focusQueue:
		return "chat"
	case focusFileNavigation, focusFileInput, focusFilePreview, focusFileConfirmation:
		return "files"
	case focusModel:
		return "model"
	case focusLanguage:
		return "language"
	case focusPermission:
		return "permission"
	case focusApproval:
		return "approval"
	}
	return "menu"
}
